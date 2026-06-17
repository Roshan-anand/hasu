const FORMATTER = new Intl.RelativeTimeFormat('en', { numeric: 'auto' });

const DIVISIONS: { amount: number; unit: Intl.RelativeTimeFormatUnit }[] = [
	{ amount: 60, unit: 'seconds' },
	{ amount: 60, unit: 'minutes' },
	{ amount: 24, unit: 'hours' },
	{ amount: 7, unit: 'days' },
	{ amount: 4.345, unit: 'weeks' },
	{ amount: 12, unit: 'months' },
	{ amount: Number.POSITIVE_INFINITY, unit: 'years' }
];

export function timeAgo(date: string | Date): string {
	if (!date) return '~~~';

	const now = Date.now();
	const then = typeof date === 'string' ? Date.parse(date) : date.getTime();
	let duration = (then - now) / 1000;

	for (const division of DIVISIONS) {
		if (Math.abs(duration) < division.amount) {
			return FORMATTER.format(Math.round(duration), division.unit);
		}
		duration /= division.amount;
	}

	return 'just now';
}

export function formatDate(date: string | Date): string {
	const d = typeof date === 'string' ? new Date(date) : date;
	return d.toLocaleDateString('en', { month: 'short', day: 'numeric', year: 'numeric' });
}
