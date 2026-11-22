// The token panel: values go to the server, which derives every partner
// colour in Go (as the theme stylesheet does), and the answer is applied to
// each frame through CSSOM, which the catalogue's strict CSP allows. Frames
// that load later (they're lazy) get the last answer when they load.
const form = document.getElementById("catalog-controls");

let pending;
let inflight;
let current = {};

function paint(root) {
	for (const [name, value] of Object.entries(current)) root.style.setProperty(name, value);
}

function paintAll() {
	paint(document.documentElement);
	for (const frame of document.querySelectorAll("iframe")) {
		if (frame.contentDocument?.documentElement) paint(frame.contentDocument.documentElement);
	}
}

function apply() {
	clearTimeout(pending);
	pending = setTimeout(async () => {
		inflight?.abort();
		inflight = new AbortController();
		const params = new URLSearchParams();
		for (const input of form.querySelectorAll("fieldset:last-of-type input")) {
			params.set(input.name, input.value + (input.dataset.unit ?? ""));
		}
		try {
			const response = await fetch("tokens.json?" + params, { signal: inflight.signal });
			current = await response.json();
			paintAll();
		} catch (err) {
			if (err.name !== "AbortError") console.error("catalog: tokens", err);
		}
	}, 80);
}

form?.addEventListener("input", (event) => {
	if (event.target.name !== "width") apply();
});

for (const frame of document.querySelectorAll("iframe")) {
	frame.addEventListener("load", () => paint(frame.contentDocument.documentElement));
}
