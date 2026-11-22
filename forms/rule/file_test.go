package rule_test

import (
	"bytes"
	"context"
	"mime/multipart"
	"slices"
	"testing"

	"github.com/epistemer/errors"
	"github.com/epistemer/ui/forms/files"
	"github.com/epistemer/ui/forms/rule"
)

func newFileHeader(t *testing.T, filename string, content []byte) *multipart.FileHeader {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	reader := multipart.NewReader(&body, writer.Boundary())
	form, err := reader.ReadForm(10 << 20)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = form.RemoveAll() })

	files := form.File["file"]
	if len(files) == 0 {
		t.Fatal("no file in multipart form")
	}
	return files[0]
}

func TestFileSizeFormatting(t *testing.T) {
	ctx := context.Background()
	emptyFile := newFileHeader(t, "empty.txt", []byte{})

	for _, tc := range []struct {
		bytes    int64
		wantText string
	}{
		{5 << 20, "must be at least 5 MB"},
		{1536, "must be at least 1536 B"},
		{2048, "must be at least 2 KB"},
		{1 << 30, "must be at least 1 GB"},
		{0, "must be at least 0 B"},
		{100, "must be at least 100 B"},
	} {
		r := rule.MinSize(tc.bytes)
		// testing on empty file of size 0 (or smaller than tc.bytes when tc.bytes > 0)
		if tc.bytes > 0 {
			err := rule.Check(ctx, emptyFile, r)
			if err == nil || err.Error() != tc.wantText {
				t.Errorf("MinSize(%d) error = %q, want %q", tc.bytes, err, tc.wantText)
			}
		}
	}
}

func TestMinSizeAndMaxSize(t *testing.T) {
	ctx := context.Background()
	file1024 := newFileHeader(t, "data.bin", make([]byte, 1024))

	t.Run("MinSize", func(t *testing.T) {
		rMin := rule.MinSize(1024)
		if err := rule.Check(ctx, file1024, rMin); err != nil {
			t.Errorf("1024-byte file should pass MinSize(1024): %v", err)
		}
		rMinHigher := rule.MinSize(2048)
		err := rule.Check(ctx, file1024, rMinHigher)
		if !errors.Is(err, rule.ErrMinSize) {
			t.Errorf("1024-byte file should fail MinSize(2048), got %v", err)
		}
		if err != nil && err.Error() != "must be at least 2 KB" {
			t.Errorf("got %q, want 'must be at least 2 KB'", err.Error())
		}
		if err := rule.Check(ctx, nil, rMin); !errors.Is(err, rule.ErrMinSize) {
			t.Errorf("nil file should fail MinSize, got %v", err)
		}
	})

	t.Run("MaxSize", func(t *testing.T) {
		rMax := rule.MaxSize(1024)
		if err := rule.Check(ctx, file1024, rMax); err != nil {
			t.Errorf("1024-byte file should pass MaxSize(1024): %v", err)
		}
		rMaxLower := rule.MaxSize(512)
		err := rule.Check(ctx, file1024, rMaxLower)
		if !errors.Is(err, rule.ErrMaxSize) {
			t.Errorf("1024-byte file should fail MaxSize(512), got %v", err)
		}
		if err != nil && err.Error() != "must be at most 512 B" {
			t.Errorf("got %q, want 'must be at most 512 B'", err.Error())
		}
		if err := rule.Check(ctx, nil, rMax); !errors.Is(err, rule.ErrMaxSize) {
			t.Errorf("nil file should fail MaxSize, got %v", err)
		}
		attrs := map[string]string{}
		for k, v := range rMax.Attrs() {
			attrs[k] = v
		}
		if attrs["data-ui-max-size"] != "1024" {
			t.Errorf("data-ui-max-size = %q, want 1024", attrs["data-ui-max-size"])
		}
	})
}

func TestContentType(t *testing.T) {
	ctx := context.Background()

	// PNG signature: \x89PNG\r\n\x1a\n followed by IHDR chunk
	pngData := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15c4")
	pngFile := newFileHeader(t, "sample.png", pngData)

	t.Run("PNG with image/*", func(t *testing.T) {
		r := rule.ContentType("image/*")
		if err := rule.Check(ctx, pngFile, r); err != nil {
			t.Errorf("PNG file should pass ContentType('image/*'): %v", err)
		}
	})

	t.Run("PNG with image/png", func(t *testing.T) {
		r := rule.ContentType("image/png")
		if err := rule.Check(ctx, pngFile, r); err != nil {
			t.Errorf("PNG file should pass ContentType('image/png'): %v", err)
		}
	})

	t.Run("PNG with application/pdf fails", func(t *testing.T) {
		r := rule.ContentType("application/pdf")
		err := rule.Check(ctx, pngFile, r)
		if !errors.Is(err, rule.ErrContentType) {
			t.Errorf("PNG file should fail ContentType('application/pdf'), got %v", err)
		}
		if err != nil && err.Error() != "must be one of these file types: application/pdf" {
			t.Errorf("error %q, want 'must be one of these file types: application/pdf'", err.Error())
		}
	})

	t.Run("Multiple types with formatting", func(t *testing.T) {
		r := rule.ContentType("application/pdf", "image/png")
		if err := rule.Check(ctx, pngFile, r); err != nil {
			t.Errorf("PNG file should pass when image/png is in allowed types: %v", err)
		}

		attrs := map[string]string{}
		for k, v := range r.Attrs() {
			attrs[k] = v
		}
		if attrs["accept"] != "application/pdf,image/png" {
			t.Errorf("accept attribute %q, want 'application/pdf,image/png'", attrs["accept"])
		}
	})

	t.Run("Nil file fails", func(t *testing.T) {
		r := rule.ContentType("image/png")
		if err := rule.Check(ctx, nil, r); !errors.Is(err, rule.ErrContentType) {
			t.Errorf("nil file should fail ContentType, got %v", err)
		}
	})

	t.Run("No types panics", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("ContentType with no types did not panic")
			}
		}()
		rule.ContentType()
	})
}

func TestContentTypeAnything(t *testing.T) {
	fh := newFileHeader(t, "a.png", []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"))
	if err := rule.Check(context.Background(), fh, rule.ContentType("*/*")); err != nil {
		t.Errorf("*/* must match any file: %v", err)
	}
}

func TestUploadRules(t *testing.T) {
	ctx := context.Background()
	pngData := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15c4")
	pngFile := newFileHeader(t, "sample.png", pngData)
	bigFile := newFileHeader(t, "big.png", make([]byte, 2048))
	stored := []files.Stored{{Key: "k1", Name: "one.png"}, {Key: "k2", Name: "two.png"}}

	t.Run("Files checks each added file and forwards attrs", func(t *testing.T) {
		r := rule.Files(rule.MaxSize(512), rule.ContentType("image/png"), rule.MaxSize(1024))
		var gotAttrs []string
		for k, v := range r.Attrs() {
			gotAttrs = append(gotAttrs, k+"="+v)
		}
		if !slices.Equal(gotAttrs, []string{"data-ui-max-size=1024", "accept=image/png"}) {
			t.Errorf("Attrs() = %v, want [data-ui-max-size=1024 accept=image/png]", gotAttrs)
		}

		// Kept files are not checked by Files; only Added files are.
		if err := rule.Check(ctx, files.Upload{Kept: stored, Added: []*multipart.FileHeader{pngFile}}, r); err != nil {
			t.Errorf("valid added file failed: %v", err)
		}
		err := rule.Check(ctx, files.Upload{Kept: stored, Added: []*multipart.FileHeader{pngFile, bigFile}}, r)
		if !errors.Is(err, rule.ErrMaxSize) {
			t.Errorf("oversized second added file: got %v, want ErrMaxSize", err)
		}
	})

	t.Run("MinFiles and MaxFiles count kept plus added", func(t *testing.T) {
		min2 := rule.MinFiles(2)
		max2 := rule.MaxFiles(2)

		// 1 kept + 0 added = 1 (< 2)
		if err := rule.Check(ctx, files.Upload{Kept: stored[:1]}, min2); !errors.Is(err, rule.ErrMinFiles) {
			t.Errorf("1 kept: got %v, want ErrMinFiles", err)
		}
		// 1 kept + 1 added = 2 (== 2)
		two := files.Upload{Kept: stored[:1], Added: []*multipart.FileHeader{pngFile}}
		if err := rule.Check(ctx, two, min2, max2); err != nil {
			t.Errorf("1 kept + 1 added: %v", err)
		}
		// 2 kept + 1 added = 3 (> 2)
		three := files.Upload{Kept: stored, Added: []*multipart.FileHeader{pngFile}}
		if err := rule.Check(ctx, three, max2); !errors.Is(err, rule.ErrMaxFiles) {
			t.Errorf("2 kept + 1 added: got %v, want ErrMaxFiles", err)
		}
	})
}
