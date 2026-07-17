/** 5×5 dot matrix grid constants and pure math helpers. */

export const MATRIX_SIZE = 5;
const CENTER = Math.floor(MATRIX_SIZE / 2);
const MAX_RADIUS = Math.hypot(CENTER, CENTER);

/** Convert (row, col) to a flat 1-D index in row-major order. */
export function rowMajorIndex(row: number, col: number): number {
	return row * MATRIX_SIZE + col;
}

/** Inverse of rowMajorIndex — returns { row, col }. */
export function indexToCoord(index: number): { row: number; col: number } {
	return {
		row: Math.floor(index / MATRIX_SIZE),
		col: index % MATRIX_SIZE
	};
}

/** Euclidean distance from the grid center. */
export function distanceFromCenter(index: number): number {
	const { row, col } = indexToCoord(index);
	return Math.hypot(row - CENTER, col - CENTER);
}

/** Polar angle (radians) from the grid center. */
export function polarAngle(index: number): number {
	const { row, col } = indexToCoord(index);
	return Math.atan2(row - CENTER, col - CENTER);
}

/** Radius normalised to [0, 1] where 1 = corner distance. */
export function normalizedRadius(index: number): number {
	const { row, col } = indexToCoord(index);
	return Math.hypot(row - CENTER, col - CENTER) / MAX_RADIUS;
}

/** Manhattan (taxicab) distance from center. */
export function manhattanDistance(index: number): number {
	const { row, col } = indexToCoord(index);
	return Math.abs(row - CENTER) + Math.abs(col - CENTER);
}
