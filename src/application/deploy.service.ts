import { DeployResult } from "../domain/contract";
import { HardhatDeployer } from "../infrastructure/blockchain/hardhat.deployer";

export class DeployService {
  constructor(private readonly deployer: HardhatDeployer) {}

  async deploy(networkName: string, verify = true): Promise<DeployResult> {
    return this.deployer.deploy(networkName, verify);
  }

  compile(): void {
    this.deployer.compile();
  }
}
