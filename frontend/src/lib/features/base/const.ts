export const getOrgProjectsQueryKey = (orgId: string) => ['project-list', 'org', orgId] as const;

export const getOrphanVolumesQueryKey = (orgId: string) =>
	['orphan-volumes', 'org', orgId, 'all'] as const;
