// The token panel: values go to the server, which derives every partner
// colour in Go (as the theme stylesheet does), and the answer is applied to
// each frame through CSSOM, which the catalogue's strict CSP allows.
const form = document.getElementById("catalog-controls");

let pending;
function apply() {
	clearTimeout(pending);
	pending = setTimeout(async () => {
		const params = new URLSearchParams();
		for (const input of form.querySelectorAll("fieldset:last-of-type input")) {
			params.set(input.name, input.value + (input.dataset.unit ?? ""));
		}
		const tokens = await (await fetch("/tokens.json?" + params)).json();
		const roots = [document.documentElement];
		for (const frame of document.querySelectorAll("iframe")) {
			if (frame.contentDocument) roots.push(frame.contentDocument.documentElement);
		}
		for (const root of roots) {
			for (const [name, value] of Object.entries(tokens)) root.style.setProperty(name, value);
		}
	}, 80);
}

form?.addEventListener("input", (event) => {
	if (event.target.name !== "width") apply();
});
