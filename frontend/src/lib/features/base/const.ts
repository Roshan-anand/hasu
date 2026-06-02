export const getOrgProjectsQueryKey = (orgId: string) => ['project-list', orgId] as const;

export const getOrphanVolumesQueryKey = (orgId: string) =>
	['orphan-volumes', 'org', orgId, 'all'] as const;

export const getOrgsQueryKey = (email: string) => ['orgs', email] as const;
