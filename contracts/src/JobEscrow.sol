// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Ag3nt} from "./Ag3nt.sol";
import {IERC20} from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import {SafeERC20} from "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import {ReentrancyGuard} from "@openzeppelin/contracts/utils/ReentrancyGuard.sol";

/// @title JobEscrow — the ag3nt job marketplace
/// @notice One escrow contract holds the state machine for every job:
///
///     Posted ──claim──> Claimed ──submit──> Submitted ──release──> Released
///        │                  │                    │
///     cancel             reclaim              dispute
///        │              (timeout)                │
///        v                  v                    v
///     Cancelled          Expired             Disputed
///
/// The buyer's payment is custodied from postJob until it settles, so neither
/// side can rug the other mid-trade. The worker stakes a bond on claim; if they
/// claim and then ghost (never submit), the buyer can reclaim after a timeout and
/// the bond is slashed to the buyer. That bond is what makes claiming have skin.
///
/// MVP-1 dispute is a no-fault unwind (payment back to buyer, bond back to worker);
/// MVP-2's jury module will arbitrate disputes and decide slashing.
contract JobEscrow is ReentrancyGuard {
    using SafeERC20 for IERC20;
    using SafeERC20 for Ag3nt;

    enum State {
        None, // 0 — job id never used
        Posted, // 1 — open, awaiting a worker
        Claimed, // 2 — a worker has staked a bond and is working
        Submitted, // 3 — result submitted, in the dispute window
        Released, // 4 — settled, worker paid (terminal)
        Disputed, // 5 — buyer disputed within window (terminal, MVP-1)
        Cancelled, // 6 — buyer cancelled before any claim (terminal)
        Expired // 7 — worker ghosted, buyer reclaimed after timeout (terminal)
    }

    struct Job {
        address buyer; // who posted and funded the job
        address worker; // who claimed it (zero until claimed)
        uint96 payment; // ag3nt paid to the worker on release
        uint96 workerBond; // ag3nt the worker staked on claim
        uint40 claimedAt; // timestamp of claim (for the claim timeout)
        uint40 submittedAt; // timestamp of submission (for the dispute window)
        State state;
        bytes32 inputHash; // commitment to the prompt/input
        bytes32 resultHash; // commitment to the result (set on submit)
        bytes result; // cleartext result (revealed on submit)
    }

    /// @notice After submission, how long the buyer has to dispute before anyone
    ///         can release payment to the worker.
    uint256 public constant DISPUTE_WINDOW = 60 seconds;

    /// @notice After claim, how long a worker has to submit before the buyer may
    ///         reclaim the payment and slash the worker's bond.
    uint256 public constant CLAIM_TIMEOUT = 1 hours;

    /// @notice Fixed protocol reward minted to the worker on a successful release,
    ///         on top of the buyer's payment. Demonstrates Bitcoin-style
    ///         issuance-for-useful-work; clamps gracefully at the token cap.
    uint256 public constant JOB_REWARD = 10 ether;

    Ag3nt public immutable token;
    uint256 public nextJobId;
    mapping(uint256 => Job) public jobs;

    event JobPosted(uint256 indexed id, address indexed buyer, uint96 payment, bytes32 inputHash);
    event JobClaimed(uint256 indexed id, address indexed worker, uint96 bond);
    event JobSubmitted(uint256 indexed id, bytes32 resultHash, bytes result);
    event JobReleased(uint256 indexed id, address indexed worker, uint96 payment, uint256 reward);
    event JobDisputed(uint256 indexed id);
    event JobCancelled(uint256 indexed id);
    event JobExpired(uint256 indexed id, address indexed worker, uint96 slashedBond);

    error WrongState(State have, State want);
    error NotBuyer();
    error NotWorker();
    error ZeroPayment();
    error WindowOpen(); // dispute window still open
    error TimeoutNotReached();

    /// @notice Deploys its own token, passing itself as the sole minter. This
    ///         resolves the chicken-and-egg between token and marketplace in a
    ///         single transaction: the escrow IS the marketplace, by construction.
    /// @param initialHolder Recipient of the operating pre-mint (Agent A).
    /// @param preMint       Amount pre-minted to `initialHolder` at deploy.
    constructor(address initialHolder, uint256 preMint) {
        token = new Ag3nt(address(this), initialHolder, preMint);
    }

    // ─────────────────────────────────────────────────────────────────────────
    // Buyer: post a job
    // ─────────────────────────────────────────────────────────────────────────

    /// @notice Post a job, locking `payment` ag3nt in escrow. Buyer must have
    ///         approved this contract for at least `payment` first.
    function postJob(uint96 payment, bytes32 inputHash) external nonReentrant returns (uint256 id) {
        if (payment == 0) revert ZeroPayment();
        token.safeTransferFrom(msg.sender, address(this), payment);

        id = ++nextJobId;
        Job storage j = jobs[id];
        j.buyer = msg.sender;
        j.payment = payment;
        j.inputHash = inputHash;
        j.state = State.Posted;

        emit JobPosted(id, msg.sender, payment, inputHash);
    }

    /// @notice Cancel a job that no worker has claimed yet; refunds the payment.
    function cancelJob(uint256 id) external nonReentrant {
        Job storage j = jobs[id];
        if (j.state != State.Posted) revert WrongState(j.state, State.Posted);
        if (msg.sender != j.buyer) revert NotBuyer();

        j.state = State.Cancelled;
        token.safeTransfer(j.buyer, j.payment);

        emit JobCancelled(id);
    }

    // ─────────────────────────────────────────────────────────────────────────
    // Worker: claim + submit
    // ─────────────────────────────────────────────────────────────────────────

    /// @notice Claim a posted job, staking `bond` ag3nt. Worker must have approved
    ///         this contract for at least `bond` first. The bond is returned on a
    ///         clean settlement and slashed only if the worker ghosts.
    function claimJob(uint256 id, uint96 bond) external nonReentrant {
        Job storage j = jobs[id];
        if (j.state != State.Posted) revert WrongState(j.state, State.Posted);

        if (bond > 0) {
            token.safeTransferFrom(msg.sender, address(this), bond);
        }
        j.worker = msg.sender;
        j.workerBond = bond;
        j.claimedAt = uint40(block.timestamp);
        j.state = State.Claimed;

        emit JobClaimed(id, msg.sender, bond);
    }

    /// @notice Submit the result. Reveals the cleartext result and commits its hash.
    function submitResult(uint256 id, bytes32 resultHash, bytes calldata result) external {
        Job storage j = jobs[id];
        if (j.state != State.Claimed) revert WrongState(j.state, State.Claimed);
        if (msg.sender != j.worker) revert NotWorker();

        j.resultHash = resultHash;
        j.result = result;
        j.submittedAt = uint40(block.timestamp);
        j.state = State.Submitted;

        emit JobSubmitted(id, resultHash, result);
    }

    // ─────────────────────────────────────────────────────────────────────────
    // Settlement
    // ─────────────────────────────────────────────────────────────────────────

    /// @notice Release payment to the worker. The buyer may call any time after
    ///         submission (early accept); anyone may call once the dispute window
    ///         has elapsed. Worker receives payment + bond back + a minted reward.
    function release(uint256 id) external nonReentrant {
        Job storage j = jobs[id];
        if (j.state != State.Submitted) revert WrongState(j.state, State.Submitted);
        if (msg.sender != j.buyer && block.timestamp < j.submittedAt + DISPUTE_WINDOW) {
            revert WindowOpen();
        }

        j.state = State.Released;
        uint96 payment = j.payment;
        uint96 bond = j.workerBond;
        address worker = j.worker;

        token.safeTransfer(worker, uint256(payment) + uint256(bond));
        uint256 minted = token.mintForJob(worker, JOB_REWARD);

        emit JobReleased(id, worker, payment, minted);
    }

    /// @notice Buyer disputes a submitted result within the dispute window. MVP-1
    ///         unwinds with no fault: payment returns to the buyer, bond to the
    ///         worker. (MVP-2's jury will arbitrate and decide slashing.)
    function dispute(uint256 id) external nonReentrant {
        Job storage j = jobs[id];
        if (j.state != State.Submitted) revert WrongState(j.state, State.Submitted);
        if (msg.sender != j.buyer) revert NotBuyer();
        if (block.timestamp >= j.submittedAt + DISPUTE_WINDOW) revert WindowOpen();

        j.state = State.Disputed;
        token.safeTransfer(j.buyer, j.payment);
        if (j.workerBond > 0) {
            token.safeTransfer(j.worker, j.workerBond);
        }

        emit JobDisputed(id);
    }

    /// @notice Buyer reclaims a claimed-but-never-submitted job after the claim
    ///         timeout. Payment refunds to the buyer and the worker's bond is
    ///         slashed to the buyer — the penalty for claiming and ghosting.
    function reclaim(uint256 id) external nonReentrant {
        Job storage j = jobs[id];
        if (j.state != State.Claimed) revert WrongState(j.state, State.Claimed);
        if (msg.sender != j.buyer) revert NotBuyer();
        if (block.timestamp < j.claimedAt + CLAIM_TIMEOUT) revert TimeoutNotReached();

        j.state = State.Expired;
        uint96 slashed = j.workerBond;
        address worker = j.worker;
        token.safeTransfer(j.buyer, uint256(j.payment) + uint256(slashed));

        emit JobExpired(id, worker, slashed);
    }

    /// @notice Read a job (the public mapping getter omits the dynamic `result`
    ///         bytes from some tooling; this returns the whole struct).
    function getJob(uint256 id) external view returns (Job memory) {
        return jobs[id];
    }
}
