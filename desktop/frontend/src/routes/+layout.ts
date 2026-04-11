// Wails apps are always browser-only — disable SSR and prerendering.
// Without this, SvelteKit's dev server does a server-side pass that hits
// `window` in the Wails runtime bindings (EventsOn, etc.) and crashes.
export const ssr = false;
export const prerender = false;
