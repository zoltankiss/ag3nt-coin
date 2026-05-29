// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Test} from "forge-std/Test.sol";
import {Ag3nt} from "../src/Ag3nt.sol";

contract Ag3ntTest is Test {
    Ag3nt token;

    address marketplace = address(0xBEEF);
    address holder = address(0xA11CE);
    address worker = address(0x4012);

    uint256 constant PRE_MINT = 1_000_000 ether;

    function setUp() public {
        // Deploy as the "marketplace" so msg.sender checks line up in these tests.
        vm.prank(marketplace);
        token = new Ag3nt(marketplace, holder, PRE_MINT);
    }

    function test_Metadata() public view {
        assertEq(token.name(), "ag3nt");
        assertEq(token.symbol(), "AGNT");
        assertEq(token.decimals(), 18);
    }

    function test_PreMintGoesToHolder() public view {
        assertEq(token.balanceOf(holder), PRE_MINT);
        assertEq(token.totalSupply(), PRE_MINT);
    }

    function test_MarketplaceAndCapSet() public view {
        assertEq(token.marketplace(), marketplace);
        assertEq(token.MAX_SUPPLY(), 21_000_000 ether);
    }

    function test_RevertWhen_PreMintExceedsCap() public {
        vm.expectRevert(Ag3nt.CapExceeded.selector);
        new Ag3nt(marketplace, holder, 21_000_001 ether);
    }

    function test_RevertWhen_ZeroMarketplace() public {
        vm.expectRevert(Ag3nt.ZeroMarketplace.selector);
        new Ag3nt(address(0), holder, PRE_MINT);
    }

    function test_MintForJob_OnlyMarketplace() public {
        vm.prank(address(0xBAD));
        vm.expectRevert(Ag3nt.OnlyMarketplace.selector);
        token.mintForJob(worker, 10 ether);
    }

    function test_MintForJob_MintsAndReturnsAmount() public {
        vm.prank(marketplace);
        uint256 minted = token.mintForJob(worker, 10 ether);
        assertEq(minted, 10 ether);
        assertEq(token.balanceOf(worker), 10 ether);
        assertEq(token.totalSupply(), PRE_MINT + 10 ether);
    }

    function test_MintForJob_ClampsAtCap() public {
        uint256 remaining = token.MAX_SUPPLY() - token.totalSupply();
        // Ask for more than remains — should mint exactly `remaining`, not revert.
        vm.prank(marketplace);
        uint256 minted = token.mintForJob(worker, remaining + 5 ether);
        assertEq(minted, remaining);
        assertEq(token.totalSupply(), token.MAX_SUPPLY());

        // Once at the cap, further mints return 0 and do nothing.
        vm.prank(marketplace);
        uint256 again = token.mintForJob(worker, 1 ether);
        assertEq(again, 0);
        assertEq(token.totalSupply(), token.MAX_SUPPLY());
    }

    function testFuzz_MintForJob_NeverExceedsCap(uint256 reward) public {
        vm.prank(marketplace);
        token.mintForJob(worker, reward);
        assertLe(token.totalSupply(), token.MAX_SUPPLY());
    }
}
