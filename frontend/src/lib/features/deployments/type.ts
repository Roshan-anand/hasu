export type DeleteDeploymentPayload = {
	deployment_id: string;
};

export type DeleteDeploymentResponse = {
	message: string;
};

export type DeploymentStatus = 'building' | 'ready' | 'error' | 'queued' | 'inactive' | 'pruned';
