// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Test} from "forge-std/Test.sol";
import {Ag3nt} from "../src/Ag3nt.sol";
import {JobEscrow} from "../src/JobEscrow.sol";

contract JobEscrowTest is Test {
    JobEscrow escrow;
    Ag3nt token;

    address buyer = address(0xA11CE); // Agent A
    address workerA = address(0x111); // Hermes-64
    address workerB = address(0x222); // Hermes-24
    address stranger = address(0x333);

    uint256 constant PRE_MINT = 1_000_000 ether;
    uint96 constant PAYMENT = 100 ether;
    uint96 constant BOND = 10 ether;
    uint96 constant WORKER_SEED = 1_000 ether;

    bytes32 constant INPUT_HASH = keccak256("ticket body to title");
    bytes RESULT = bytes("Fix login redirect loop on Safari");

    function setUp() public {
        // Escrow deploys the token and pre-mints to the buyer (Agent A).
        escrow = new JobEscrow(buyer, PRE_MINT);
        token = escrow.token();

        // Seed both workers with ag3nt so they can stake bonds.
        vm.startPrank(buyer);
        token.transfer(workerA, WORKER_SEED);
        token.transfer(workerB, WORKER_SEED);
        vm.stopPrank();

        // Everyone approves the escrow to move their tokens.
        vm.prank(buyer);
        token.approve(address(escrow), type(uint256).max);
        vm.prank(workerA);
        token.approve(address(escrow), type(uint256).max);
        vm.prank(workerB);
        token.approve(address(escrow), type(uint256).max);
    }

    // ── helpers ───────────────────────────────────────────────────────────────

    function _post() internal returns (uint256 id) {
        vm.prank(buyer);
        id = escrow.postJob(PAYMENT, INPUT_HASH);
    }

    function _postAndClaim(address worker) internal returns (uint256 id) {
        id = _post();
        vm.prank(worker);
        escrow.claimJob(id, BOND);
    }

    function _postClaimSubmit(address worker) internal returns (uint256 id) {
        id = _postAndClaim(worker);
        vm.prank(worker);
        escrow.submitResult(id, keccak256(RESULT), RESULT);
    }

    // ── post / cancel ───────────────────────────────────────────────────────

    function test_PostJob_LocksPayment() public {
        uint256 before = token.balanceOf(buyer);
        uint256 id = _post();

        JobEscrow.Job memory j = escrow.getJob(id);
        assertEq(id, 1);
        assertEq(uint8(j.state), uint8(JobEscrow.State.Posted));
        assertEq(j.buyer, buyer);
        assertEq(j.payment, PAYMENT);
        assertEq(j.inputHash, INPUT_HASH);
        assertEq(token.balanceOf(address(escrow)), PAYMENT);
        assertEq(token.balanceOf(buyer), before - PAYMENT);
    }

    function test_RevertWhen_PostZeroPayment() public {
        vm.prank(buyer);
        vm.expectRevert(JobEscrow.ZeroPayment.selector);
        escrow.postJob(0, INPUT_HASH);
    }

    function test_CancelJob_RefundsBuyer() public {
        uint256 before = token.balanceOf(buyer);
        uint256 id = _post();
        vm.prank(buyer);
        escrow.cancelJob(id);

        assertEq(uint8(escrow.getJob(id).state), uint8(JobEscrow.State.Cancelled));
        assertEq(token.balanceOf(buyer), before);
        assertEq(token.balanceOf(address(escrow)), 0);
    }

    function test_RevertWhen_CancelNotBuyer() public {
        uint256 id = _post();
        vm.prank(stranger);
        vm.expectRevert(JobEscrow.NotBuyer.selector);
        escrow.cancelJob(id);
    }

    function test_RevertWhen_CancelAfterClaim() public {
        uint256 id = _postAndClaim(workerA);
        vm.prank(buyer);
        vm.expectRevert(
            abi.encodeWithSelector(JobEscrow.WrongState.selector, JobEscrow.State.Claimed, JobEscrow.State.Posted)
        );
        escrow.cancelJob(id);
    }

    // ── claim / submit ────────────────────────────────────────────────────────

    function test_ClaimJob_StakesBond() public {
        uint256 id = _post();
        uint256 before = token.balanceOf(workerA);
        vm.prank(workerA);
        escrow.claimJob(id, BOND);

        JobEscrow.Job memory j = escrow.getJob(id);
        assertEq(uint8(j.state), uint8(JobEscrow.State.Claimed));
        assertEq(j.worker, workerA);
        assertEq(j.workerBond, BOND);
        assertEq(token.balanceOf(workerA), before - BOND);
        assertEq(token.balanceOf(address(escrow)), uint256(PAYMENT) + BOND);
    }

    function test_RevertWhen_DoubleClaim() public {
        uint256 id = _postAndClaim(workerA);
        vm.prank(workerB);
        vm.expectRevert(
            abi.encodeWithSelector(JobEscrow.WrongState.selector, JobEscrow.State.Claimed, JobEscrow.State.Posted)
        );
        escrow.claimJob(id, BOND);
    }

    function test_SubmitResult_RevealsResult() public {
        uint256 id = _postAndClaim(workerA);
        vm.prank(workerA);
        escrow.submitResult(id, keccak256(RESULT), RESULT);

        JobEscrow.Job memory j = escrow.getJob(id);
        assertEq(uint8(j.state), uint8(JobEscrow.State.Submitted));
        assertEq(j.resultHash, keccak256(RESULT));
        assertEq(string(j.result), string(RESULT));
        assertEq(j.submittedAt, uint40(block.timestamp));
    }

    function test_RevertWhen_SubmitNotWorker() public {
        uint256 id = _postAndClaim(workerA);
        vm.prank(workerB);
        vm.expectRevert(JobEscrow.NotWorker.selector);
        escrow.submitResult(id, keccak256(RESULT), RESULT);
    }

    // ── release ─────────────────────────────────────────────────────────────

    function test_Release_EarlyByBuyer_PaysWorkerWithReward() public {
        uint256 id = _postClaimSubmit(workerA);
        uint256 workerBefore = token.balanceOf(workerA); // already down by BOND
        uint256 supplyBefore = token.totalSupply();

        vm.prank(buyer); // early accept, inside the window
        escrow.release(id);

        JobEscrow.Job memory j = escrow.getJob(id);
        assertEq(uint8(j.state), uint8(JobEscrow.State.Released));
        // worker gets payment + bond back + freshly minted reward
        assertEq(token.balanceOf(workerA), workerBefore + PAYMENT + BOND + escrow.JOB_REWARD());
        assertEq(token.totalSupply(), supplyBefore + escrow.JOB_REWARD());
        assertEq(token.balanceOf(address(escrow)), 0);
    }

    function test_Release_ByAnyoneAfterWindow() public {
        uint256 id = _postClaimSubmit(workerA);
        vm.warp(block.timestamp + escrow.DISPUTE_WINDOW());
        // a third party can settle once the window has elapsed
        vm.prank(stranger);
        escrow.release(id);
        assertEq(uint8(escrow.getJob(id).state), uint8(JobEscrow.State.Released));
    }

    function test_RevertWhen_ReleaseByStrangerWithinWindow() public {
        uint256 id = _postClaimSubmit(workerA);
        vm.prank(stranger);
        vm.expectRevert(JobEscrow.WindowOpen.selector);
        escrow.release(id);
    }

    function test_RevertWhen_ReleaseBeforeSubmit() public {
        uint256 id = _postAndClaim(workerA);
        vm.prank(buyer);
        vm.expectRevert(
            abi.encodeWithSelector(JobEscrow.WrongState.selector, JobEscrow.State.Claimed, JobEscrow.State.Submitted)
        );
        escrow.release(id);
    }

    // ── dispute ─────────────────────────────────────────────────────────────

    function test_Dispute_RefundsBuyerReturnsBond() public {
        uint256 buyerStart = token.balanceOf(buyer);
        uint256 workerStart = token.balanceOf(workerA);
        uint256 id = _postClaimSubmit(workerA);

        vm.prank(buyer);
        escrow.dispute(id);

        JobEscrow.Job memory j = escrow.getJob(id);
        assertEq(uint8(j.state), uint8(JobEscrow.State.Disputed));
        // no-fault unwind: both sides made whole, no mint
        assertEq(token.balanceOf(buyer), buyerStart);
        assertEq(token.balanceOf(workerA), workerStart);
        assertEq(token.balanceOf(address(escrow)), 0);
    }

    function test_RevertWhen_DisputeAfterWindow() public {
        uint256 id = _postClaimSubmit(workerA);
        vm.warp(block.timestamp + escrow.DISPUTE_WINDOW());
        vm.prank(buyer);
        vm.expectRevert(JobEscrow.WindowOpen.selector);
        escrow.dispute(id);
    }

    function test_RevertWhen_DisputeNotBuyer() public {
        uint256 id = _postClaimSubmit(workerA);
        vm.prank(stranger);
        vm.expectRevert(JobEscrow.NotBuyer.selector);
        escrow.dispute(id);
    }

    // ── reclaim (worker ghosted) ───────────────────────────────────────────────

    function test_Reclaim_SlashesBondAfterTimeout() public {
        uint256 buyerStart = token.balanceOf(buyer);
        uint256 id = _postAndClaim(workerA); // claimed, never submitted
        vm.warp(block.timestamp + escrow.CLAIM_TIMEOUT());

        vm.prank(buyer);
        escrow.reclaim(id);

        JobEscrow.Job memory j = escrow.getJob(id);
        assertEq(uint8(j.state), uint8(JobEscrow.State.Expired));
        // buyer recovers payment AND the slashed bond
        assertEq(token.balanceOf(buyer), buyerStart + BOND);
        assertEq(token.balanceOf(address(escrow)), 0);
    }

    function test_RevertWhen_ReclaimBeforeTimeout() public {
        uint256 id = _postAndClaim(workerA);
        vm.prank(buyer);
        vm.expectRevert(JobEscrow.TimeoutNotReached.selector);
        escrow.reclaim(id);
    }

    function test_RevertWhen_ReclaimAfterSubmit() public {
        uint256 id = _postClaimSubmit(workerA);
        vm.warp(block.timestamp + escrow.CLAIM_TIMEOUT());
        vm.prank(buyer);
        vm.expectRevert(
            abi.encodeWithSelector(JobEscrow.WrongState.selector, JobEscrow.State.Submitted, JobEscrow.State.Claimed)
        );
        escrow.reclaim(id);
    }

    // ── two workers racing ────────────────────────────────────────────────────

    function test_TwoWorkersRace_OnlyFirstClaims() public {
        uint256 id = _post();
        vm.prank(workerA);
        escrow.claimJob(id, BOND); // A wins
        vm.prank(workerB);
        vm.expectRevert(
            abi.encodeWithSelector(JobEscrow.WrongState.selector, JobEscrow.State.Claimed, JobEscrow.State.Posted)
        );
        escrow.claimJob(id, BOND); // B too late
        assertEq(escrow.getJob(id).worker, workerA);
    }
}
