// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "forge-std/Test.sol";
import "../contracts/ForwarderFactory.sol";
import "../contracts/Forwarder.sol";
import "../contracts/MockBEP20.sol";

contract ForwarderFactoryTest is Test {
    ForwarderFactory factory;
    MockBEP20 token;

    address owner;
    address relayer;
    address motherWallet;
    address attacker;
    address user;

    uint256 constant USER_ID = 1;

    function setUp() public {
        owner = makeAddr("owner");
        relayer = makeAddr("relayer");
        motherWallet = makeAddr("mother");
        attacker = makeAddr("attacker");
        user = makeAddr("user");

        vm.prank(owner);
        factory = new ForwarderFactory(motherWallet, relayer);

        token = new MockBEP20("Mock USDT", "mUSDT");
    }

    function test_getAddress_predictableBeforeDeploy() public view {
        address predicted = factory.getAddress(USER_ID);
        assertTrue(predicted != address(0));
        assertEq(predicted.code.length, 0);
    }

    function test_getAddress_differentUserIds() public view {
        address addr1 = factory.getAddress(1);
        address addr2 = factory.getAddress(2);
        assertTrue(addr1 != addr2);
    }

    function test_deployAndSweepNative_sweepsToMother() public {
        address userWallet = factory.getAddress(USER_ID);
        vm.deal(user, 1 ether);
        vm.prank(user);
        (bool ok,) = userWallet.call{value: 1 ether}("");
        assertTrue(ok);

        uint256 before = motherWallet.balance;
        vm.prank(relayer);
        factory.deployAndSweepNative(USER_ID);
        assertEq(motherWallet.balance - before, 1 ether);
    }

    function test_deployAndSweepNative_secondSweepWithoutRedeploy() public {
        address userWallet = factory.getAddress(USER_ID);
        vm.deal(user, 1.5 ether);

        vm.prank(user);
        (bool ok,) = userWallet.call{value: 1 ether}("");
        assertTrue(ok);
        vm.prank(relayer);
        factory.deployAndSweepNative(USER_ID);

        vm.prank(user);
        (ok,) = userWallet.call{value: 0.5 ether}("");
        assertTrue(ok);

        vm.prank(relayer);
        vm.recordLogs();
        factory.deployAndSweepNative(USER_ID);
        Vm.Log[] memory entries = vm.getRecordedLogs();
        for (uint256 i = 0; i < entries.length; i++) {
            assertFalse(
                entries[i].topics[0] == keccak256("WalletDeployed(uint256,address)")
            );
        }
    }

    function test_deployAndSweepNative_revertsForNonRelayer() public {
        vm.prank(attacker);
        vm.expectRevert("Factory: not relayer");
        factory.deployAndSweepNative(USER_ID);
    }

    function test_deployAndSweepToken_sweepsToMother() public {
        address userWallet = factory.getAddress(USER_ID);
        token.mint(userWallet, 100 ether);

        uint256 before = token.balanceOf(motherWallet);
        vm.prank(relayer);
        factory.deployAndSweepToken(USER_ID, address(token));
        assertEq(token.balanceOf(motherWallet) - before, 100 ether);
    }

    function test_emergencyWithdrawNative_ownerCanRescue() public {
        address userWallet = factory.getAddress(USER_ID);
        vm.deal(user, 1 ether);
        vm.prank(user);
        (bool ok,) = userWallet.call{value: 1 ether}("");
        assertTrue(ok);

        vm.prank(relayer);
        factory.deployWallet(USER_ID);

        uint256 before = motherWallet.balance;
        vm.prank(owner);
        Forwarder(payable(userWallet)).emergencyWithdrawNative();
        assertEq(motherWallet.balance - before, 1 ether);
    }

    function test_emergencyWithdrawNative_revertsForNonOwner() public {
        address userWallet = factory.getAddress(USER_ID);
        vm.deal(user, 1 ether);
        vm.prank(user);
        (bool ok,) = userWallet.call{value: 1 ether}("");
        assertTrue(ok);

        vm.prank(relayer);
        factory.deployWallet(USER_ID);

        vm.prank(attacker);
        vm.expectRevert("Factory: not owner");
        factory.emergencyWithdrawNative(USER_ID);
    }

    function test_timelock_immediateChangeFails() public {
        vm.startPrank(owner);
        factory.requestMotherWalletChange(attacker);
        vm.expectRevert("Factory: timelock active");
        factory.applyMotherWalletChange();
        vm.stopPrank();
    }

    function test_timelock_appliesAfter48Hours() public {
        vm.startPrank(owner);
        factory.requestMotherWalletChange(attacker);
        vm.warp(block.timestamp + 48 hours + 1);
        factory.applyMotherWalletChange();
        vm.stopPrank();
        assertEq(factory.motherWallet(), attacker);
    }

    function test_timelock_cancelPendingChange() public {
        vm.startPrank(owner);
        factory.requestMotherWalletChange(attacker);
        factory.cancelMotherWalletChange();
        vm.stopPrank();
        assertEq(factory.pendingMotherWallet(), address(0));
    }

    function test_updateRelayer_onlyOwner() public {
        vm.prank(attacker);
        vm.expectRevert("Factory: not owner");
        factory.updateRelayer(attacker);

        vm.prank(owner);
        factory.updateRelayer(attacker);
        assertEq(factory.relayer(), attacker);
    }
}
