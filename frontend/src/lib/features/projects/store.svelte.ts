import { getContext, setContext } from 'svelte';

interface ProjectState {
	createDialogOpen: boolean;
	projectName: string;
	projectDescription: string;
	deletingProjectId: string;
	closeDialog: () => void;
}

class ProjectStateClass implements ProjectState {
	createDialogOpen = $state(false);
	projectName = $state('');
	projectDescription = $state('');
	deletingProjectId = $state('');
	closeDialog = () => {
		this.createDialogOpen = false;
		this.projectName = '';
		this.projectDescription = '';
	};
}

const DEFAULT_KEY = 'projects:feature:state';

export const getProjectState = (key: string = DEFAULT_KEY) => {
	return getContext<ProjectState>(Symbol.for(key));
};

export const setProjectState = (key: string = DEFAULT_KEY) => {
	setContext(Symbol.for(key), new ProjectStateClass());
};
