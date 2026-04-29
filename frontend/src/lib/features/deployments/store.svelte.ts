import { getContext, setContext } from 'svelte';

interface DeploymentsFeatureState {
	deletingDeploymentId: string;
	setDeletingDeploymentId: (id: string) => void;
}

class DeploymentsFeatureStateClass implements DeploymentsFeatureState {
	deletingDeploymentId = $state('');
	setDeletingDeploymentId = (id: string) => (this.deletingDeploymentId = id);
}

const DEFAULT_KEY = 'deployments:feature:state';

export const getDeploymentsFeatureState = (key: string = DEFAULT_KEY) => {
	return getContext<DeploymentsFeatureState>(Symbol.for(key));
};

export const setDeploymentsFeatureState = (key: string = DEFAULT_KEY) => {
	setContext(Symbol.for(key), new DeploymentsFeatureStateClass());
};
