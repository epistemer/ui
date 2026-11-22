package jstest

import "testing"

func TestFmtSinglePass(t *testing.T) {
	for _, script := range []string{"../../theme/js/repeat.js", "../../theme/js/upload.js"} {
		vm := load(t, script)
		res, err := vm.RunString(`
			[
				fmt("Item %d added", 3),
				fmt("Remove %s", "100%s.png"),
				fmt("%s is larger than 5 MB", "price$10$&_receipt.pdf"),
				fmt("No verb here", "ignored"),
			]
		`)
		if err != nil {
			t.Fatalf("%s: %v", script, err)
		}
		got := res.Export().([]any)
		want := []string{
			"Item 3 added",
			"Remove 100%s.png",
			"price$10$&_receipt.pdf is larger than 5 MB",
			"No verb here",
		}
		for i, w := range want {
			if got[i] != w {
				t.Errorf("%s: fmt[%d] = %q, want %q", script, i, got[i], w)
			}
		}
	}
}

func TestRepeatPathAndWhenRewriting(t *testing.T) {
	vm := load(t, "../../theme/js/repeat.js")
	res, err := vm.RunString(`
		[
			rewritePath("lines[__i__].sku", "lines[__i__]", "lines[2]"),
			rewritePath("lines[__i__].extras[__i__].note", "lines[__i__]", "lines[2]"),
			rewritePath("other[__i__].sku", "lines[__i__]", "lines[2]"),
			rewriteWhen(
				'{"typ":"and","args":[{"typ":"c","k":"lines[__i__].qty","op":"filled"},{"typ":"c","k":"lines[__i__].extras[__i__].note","op":"eq","v":"lines[__i__]"}]}',
				"lines[__i__]",
				"lines[4]"
			),
		]
	`)
	if err != nil {
		t.Fatal(err)
	}
	got := res.Export().([]any)
	want := []string{
		"lines[2].sku",
		"lines[2].extras[__i__].note",
		"other[__i__].sku",
		`{"typ":"and","args":[{"typ":"c","k":"lines[4].qty","op":"filled"},{"typ":"c","k":"lines[4].extras[__i__].note","op":"eq","v":"lines[4]"}]}`,
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("case %d: got %q, want %q", i, got[i], w)
		}
	}
}

func TestUploadSizeAndAccept(t *testing.T) {
	vm := load(t, "../../theme/js/upload.js")
	res, err := vm.RunString(`
		[
			size(0),
			size(512),
			size(1024),
			size(1536),
			size(1048576),
			size(5242880),
			fitsAccept("photo.PNG", "IMAGE/PNG", "image/png, image/jpeg, .pdf"),
			fitsAccept("camera.jpg", "image/webp", "image/*"),
			fitsAccept("scan.PDF", "application/octet-stream", "image/png, .pdf"),
			fitsAccept("mystery.bin", "", "image/png, application/pdf"),
			fitsAccept("mystery.bin", "", ".pdf, .png"),
			fitsAccept("notes.txt", "text/plain", "image/*, .pdf"),
		]
	`)
	if err != nil {
		t.Fatal(err)
	}
	got := res.Export().([]any)
	want := []any{
		"0 B",
		"512 B",
		"1 KB",
		"2 KB",
		"1.0 MB",
		"5.0 MB",
		true,
		true,
		true,
		true,  // untyped file defers to server when accept has only MIME types
		false, // untyped file checked by extension when accept has extensions
		false,
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("case %d: got %v, want %v", i, got[i], w)
		}
	}
}
