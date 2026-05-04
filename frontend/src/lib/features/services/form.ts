import type { GitProviderKey } from './type';

export type AppGitFormValues = {
	git_provider: GitProviderKey | '';
	git_app_id: string;
	git_repo_id: string;
	build_path: string;
	watch_path: string;
};

export const normalizePathValue = (value: string) => {
	const path = value.trim();
	return path === '' ? '/' : path;
};

export const validateAppGitForm = (value: AppGitFormValues) => {
	const fields: Record<string, string> = {};

	if (value.git_provider === '') fields.git_provider = 'Git provider is required';
	if (value.git_provider === 'github') {
		if (value.git_app_id === '') fields.git_app_id = 'GitHub app is required';
		if (value.git_repo_id === '') fields.git_repo_id = 'Repository is required';
		if (value.build_path.trim() === '') fields.build_path = 'Build path is required';
		if (value.watch_path.trim() === '') fields.watch_path = 'Watch path is required';
	}

	return Object.keys(fields).length > 0 ? fields : null;
};
