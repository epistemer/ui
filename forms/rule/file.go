package rule

import (
	"context"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"slices"
	"strings"
)

// MinSize checks that the uploaded file's size is at least bytes.
// The error argument is formatted using the largest whole unit of B, KB, MB, or GB.
func MinSize(bytes int64) Rule[*multipart.FileHeader] {
	mustNotBeNegative("MinSize", bytes)
	formatted := formatFileSize(bytes)
	return Rule[*multipart.FileHeader]{
		check: func(_ context.Context, fh *multipart.FileHeader) error {
			if fh == nil || fh.Size < bytes {
				return ErrMinSize.With(formatted)
			}
			return nil
		},
	}
}

// MaxSize checks that the uploaded file's size is at most bytes.
// The error argument is formatted using the largest whole unit of B, KB, MB, or GB.
func MaxSize(bytes int64) Rule[*multipart.FileHeader] {
	mustNotBeNegative("MaxSize", bytes)
	formatted := formatFileSize(bytes)
	return Rule[*multipart.FileHeader]{
		check: func(_ context.Context, fh *multipart.FileHeader) error {
			if fh == nil || fh.Size > bytes {
				return ErrMaxSize.With(formatted)
			}
			return nil
		},
	}
}

// ContentType sniffs up to 512 bytes of the file and validates its media type against types.
// "image/*" matches any image/ subtype. ContentType panics if no types are provided.
// It implies the accept attribute.
func ContentType(types ...string) Rule[*multipart.FileHeader] {
	if len(types) == 0 {
		panic("rule: ContentType with no types")
	}
	msgArg := strings.Join(types, ", ")
	acceptAttr := strings.Join(types, ",")
	allowed := slices.Clone(types)

	return Rule[*multipart.FileHeader]{
		check: func(_ context.Context, fh *multipart.FileHeader) error {
			if fh == nil {
				return ErrContentType.With(msgArg)
			}
			f, err := fh.Open()
			if err != nil {
				return ErrContentType.With(msgArg)
			}
			defer f.Close()

			buf := make([]byte, 512)
			n, err := io.ReadFull(f, buf)
			if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
				return ErrContentType.With(msgArg)
			}

			detected := http.DetectContentType(buf[:n])
			mediaType, _, err := mime.ParseMediaType(detected)
			if err != nil {
				mediaType = strings.TrimSpace(strings.ToLower(strings.Split(detected, ";")[0]))
			}

			for _, t := range allowed {
				t = strings.TrimSpace(strings.ToLower(t))
				if t == "*/*" {
					return nil
				}
				if strings.HasSuffix(t, "/*") {
					prefix := strings.TrimSuffix(t, "*")
					if strings.HasPrefix(mediaType, prefix) {
						return nil
					}
					continue
				}
				targetType, _, err := mime.ParseMediaType(t)
				if err != nil {
					targetType = t
				}
				if mediaType == targetType {
					return nil
				}
			}
			return ErrContentType.With(msgArg)
		},
	}.with("accept", acceptAttr)
}

func formatFileSize(b int64) string {
	const (
		gb = 1024 * 1024 * 1024
		mb = 1024 * 1024
		kb = 1024
	)
	switch {
	case b > 0 && b%gb == 0:
		return fmt.Sprintf("%d GB", b/gb)
	case b > 0 && b%mb == 0:
		return fmt.Sprintf("%d MB", b/mb)
	case b > 0 && b%kb == 0:
		return fmt.Sprintf("%d KB", b/kb)
	default:
		return fmt.Sprintf("%d B", b)
	}
}
