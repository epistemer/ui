// The forms condition evaluator (forms-design §7.3): the server's rules,
// in the browser. A pure function of a condition and a form's state, so
// the conformance vectors in forms/testdata/cond run it as they run Go.
// Written in plain ES2017 so a pure-Go engine can run it in go test.

// What strconv.ParseBool reads as false.
const falseWords = ["0", "f", "F", "false", "FALSE", "False"];

// evaluate reports whether cond holds. env describes the form:
//   env.kind(name)   "checkbox" for a checkbox, anything else for text
//   env.values(name) the submitted values, in order (an array of strings)
//   env.when(name)   the conditions on name and every node around it,
//                    which must all hold for it to be active (an array)
export function evaluate(cond, env) {
	const memo = new Map();
	const visiting = new Set();

	function active(name) {
		if (memo.has(name)) return memo.get(name);
		if (visiting.has(name)) return false; // the server rejects cycles; never loop
		visiting.add(name);
		const conds = env.when(name) || [];
		let ok = true;
		for (let i = 0; i < conds.length && ok; i++) ok = holds(conds[i]);
		visiting.delete(name);
		memo.set(name, ok);
		return ok;
	}

	function last(name) {
		const vs = env.values(name) || [];
		return vs.length ? String(vs[vs.length - 1]).trim() : "";
	}

	function ticked(name) {
		const v = last(name);
		return v !== "" && falseWords.indexOf(v) < 0;
	}

	function holds(node) {
		if (node.typ === "c") {
			if (!active(node.k)) return false;
			const box = env.kind(node.k) === "checkbox";
			if (node.op === "eq") {
				if (box) return String(ticked(node.k)) === node.v;
				const v = last(node.k);
				return v !== "" && v === node.v;
			}
			if (node.op === "filled") return box ? ticked(node.k) : last(node.k) !== "";
			throw new Error("forms: unknown condition op " + node.op);
		}
		const args = node.args || [];
		switch (node.op) {
			case "and":
				return args.every(holds);
			case "or":
				return args.some(holds);
			case "not":
				return !holds(args[0]);
		}
		throw new Error("forms: unknown condition op " + node.op);
	}

	return holds(cond);
}
