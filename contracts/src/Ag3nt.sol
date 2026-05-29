// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {ERC20} from "@openzeppelin/contracts/token/ERC20/ERC20.sol";

/// @title ag3nt — the agent-economy token
/// @notice Standard ERC-20 with a Bitcoin-style 21M hard cap. The ONLY address
///         allowed to mint new supply (beyond the constructor pre-mint) is the
///         marketplace (JobEscrow), which mints a small reward to workers when a
///         job settles successfully. Mint authority on the marketplace — not the
///         deployer — gives credible neutrality: nobody can mint outside the rules.
contract Ag3nt is ERC20 {
    /// @notice Hard cap. 21,000,000 tokens * 10^18 (18 decimals, the ERC-20 default).
    uint256 public constant MAX_SUPPLY = 21_000_000 ether;

    /// @notice The only contract permitted to mint job rewards. Set once at deploy.
    address public immutable marketplace;

    error CapExceeded();
    error OnlyMarketplace();
    error ZeroMarketplace();

    /// @param _marketplace  Address of the JobEscrow (the sole minter).
    /// @param initialHolder Seed recipient of the operating pre-mint (e.g. Agent A).
    /// @param preMint       Amount pre-minted at deploy. Must be <= MAX_SUPPLY.
    constructor(address _marketplace, address initialHolder, uint256 preMint)
        ERC20("ag3nt", "AGNT")
    {
        if (_marketplace == address(0)) revert ZeroMarketplace();
        if (preMint > MAX_SUPPLY) revert CapExceeded();
        marketplace = _marketplace;
        if (preMint > 0) {
            _mint(initialHolder, preMint); // seed operating balance
        }
    }

    /// @notice Mint a job reward to a worker. Called by the marketplace on a
    ///         successful settlement. Mints up to the remaining cap and returns
    ///         the amount actually minted — it clamps at the cap rather than
    ///         reverting, so a settlement near the cap still succeeds (the worker
    ///         always gets their escrowed payment regardless of the mint).
    /// @return minted The amount actually minted (0 once the cap is reached).
    function mintForJob(address worker, uint256 reward) external returns (uint256 minted) {
        if (msg.sender != marketplace) revert OnlyMarketplace();
        uint256 remaining = MAX_SUPPLY - totalSupply();
        minted = reward > remaining ? remaining : reward;
        if (minted > 0) {
            _mint(worker, minted);
        }
    }
}
