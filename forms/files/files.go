// Package files holds the stored-file and upload value types of forms and
// its rules.
package files

import "mime/multipart"

// Stored is a file the record already has.
type Stored struct {
	Key   string                    // the app's key, posted back to keep it
	Name  string                    // shown
	Href  interface{ URL() string } // where it opens; nil for none
	Image bool                      // Href is an image: shown as a thumbnail
}

// Upload is an upload field's value.
type Upload struct {
	Kept  []Stored                // stored files still kept, in the order given
	Added []*multipart.FileHeader // files chosen now
}
