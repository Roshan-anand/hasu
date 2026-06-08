import type { GitProviderKey, GitProviderOption, NavItem } from './type';

export const ServiceTypes = [
	{ value: 'app' as const, label: 'App Service' },
	{ value: 'psql' as const, label: 'PostgreSQL Service' }
];

export const GitProvidersList: Map<GitProviderKey, GitProviderOption> = new Map([
	[
		'github',
		{
			name: 'Github',
			icon: 'meteor-icons:github',
			listApi: '/provider/github/repo/list',
			createApi: '/api/provider/github/app/create'
		}
	],
	['gitlab', { name: 'GitLab', icon: 'material-icon-theme:gitlab', listApi: '', createApi: '' }],
	[
		'bitbucket',
		{ name: 'BitBucket', icon: 'material-icon-theme:bitbucket', listApi: '', createApi: '' }
	]
]);

export const NavItems: NavItem[] = [
	{ label: 'General', tab: '' },
	{ label: 'Deployment', tab: 'deployment' },
	{ label: 'Environment', tab: 'env' },
	{ label: 'Settings', tab: 'settings' }
];
