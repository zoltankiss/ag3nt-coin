// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Script, console} from "forge-std/Script.sol";
import {JobEscrow} from "../src/JobEscrow.sol";

/// @notice Deploys the marketplace, which in turn deploys the token and pre-mints
///         the operating balance to Agent A. One broadcast, two contracts.
///
/// Env:
///   AGENT_A_ADDRESS  — recipient of the pre-mint (the buyer wallet)
///   PRE_MINT_WEI     — pre-mint amount in wei (default 1,000,000 ether)
///
/// Run (Base Sepolia):
///   forge script script/Deploy.s.sol --rpc-url base_sepolia --broadcast --verify
contract Deploy is Script {
    function run() external {
        address agentA = vm.envAddress("AGENT_A_ADDRESS");
        uint256 preMint = vm.envOr("PRE_MINT_WEI", uint256(1_000_000 ether));

        vm.startBroadcast();
        JobEscrow escrow = new JobEscrow(agentA, preMint);
        vm.stopBroadcast();

        console.log("JobEscrow :", address(escrow));
        console.log("Ag3nt     :", address(escrow.token()));
        console.log("Agent A   :", agentA);
        console.log("Pre-mint  :", preMint);
    }
}
