export const getOrgProjectsQueryKey = (orgId: string) => ['project-list', orgId] as const;

export const getOrphanVolumesQueryKey = (orgId: string) =>
	['orphan-volumes', orgId, 'all'] as const;

export const getOrgsQueryKey = (email: string) => ['orgss', email] as const;

export const getAllInstanceQueryKey = (project: string) => ['all-instance', project] as const;
