import * as dotenv from "dotenv";
dotenv.config();

import { EnvRepository } from "../infrastructure/env/env.repository";
import { ProviderFactory } from "../infrastructure/blockchain/provider.factory";
import { HardhatDeployer } from "../infrastructure/blockchain/hardhat.deployer";
import { ContractGateway } from "../infrastructure/blockchain/contract.gateway";
import { WalletService } from "../application/wallet.service";
import { DeployService } from "../application/deploy.service";
import { ContractService } from "../application/contract.service";
import { MonitorService } from "../monitoring";

const envRepo = new EnvRepository();
envRepo.reload();

const providerFactory = new ProviderFactory(envRepo);
const hardhatDeployer = new HardhatDeployer(envRepo, providerFactory);
const contractGateway = new ContractGateway(envRepo, providerFactory);

export const walletService = new WalletService(envRepo, providerFactory);
export const deployService = new DeployService(hardhatDeployer);
export const contractService = new ContractService(contractGateway);
export const monitorService = new MonitorService(envRepo, providerFactory, contractService);
