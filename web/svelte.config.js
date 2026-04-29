import adapter from '@sveltejs/adapter-static';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	kit: {
		adapter: adapter({
			pages: '../go/internal/gui/frontend',
			assets: '../go/internal/gui/frontend',
			fallback: 'index.html',
			precompress: false
		})
	}
};

export default config;
