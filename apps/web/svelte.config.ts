import adapter from '@sveltejs/adapter-static';
import { relative, sep } from 'node:path';
import type { Config } from '@sveltejs/kit';

const config: Config = {
	compilerOptions: {
		// defaults to rune mode for the project, execept for `node_modules`. Can be removed in svelte 6.
		runes: ({ filename }) => {
			const relativePath = relative(import.meta.dirname, filename);
			const pathSegments = relativePath.toLowerCase().split(sep);
			const isExternalLibrary = pathSegments.includes('node_modules');

			return isExternalLibrary ? undefined : true;
		}
	},
	kit: {
		// adapter-auto only supports some environments, see https://svelte.dev/docs/kit/adapter-auto for a list.
		// If your environment is not supported, or you settled on a specific environment, switch out the adapter.
		// See https://svelte.dev/docs/kit/adapters for more information about adapters.
		adapter: adapter({
			pages: process.env.STATIC_OUT_PATH || '../server/frontend/dist',
			assets: process.env.STATIC_OUT_PATH || '../server/frontend/dist',
			fallback: 'not-found.html'
		}),
		alias: {
			'@/*': './src/lib/*'
		},
		router: {
			type: 'hash'
		}
	}
};

export default config;
