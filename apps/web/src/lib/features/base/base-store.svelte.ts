import { getContext, setContext } from 'svelte';

interface BaseState {
	inlinePanelDrawer: boolean;
	setPanelDrawerState: (state: boolean) => void;
}

class BaseStateClass implements BaseState {
	inlinePanelDrawer: boolean = $state(false);

	setPanelDrawerState = (state: boolean) => (this.inlinePanelDrawer = state);
}

const DEFAULT_KEY = 'base:state';

export const getBaseState = (key: string = DEFAULT_KEY) => getContext<BaseState>(Symbol.for(key));

export const setBaseState = (key: string = DEFAULT_KEY) => {
	setContext(Symbol.for(key), new BaseStateClass());
};
