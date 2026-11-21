package uitest_test

import (
	"strings"
	"testing"

	"github.com/epistemer/ui/uitest"
)

func TestNormalise(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "attribute order independence",
			input: `<div c="3" a="1" b="2"></div>`,
			want:  "<div a=\"1\" b=\"2\" c=\"3\"></div>\n",
		},
		{
			name:  "attribute order matches different input order",
			input: `<div b="2" c="3" a="1"></div>`,
			want:  "<div a=\"1\" b=\"2\" c=\"3\"></div>\n",
		},
		{
			name:  "text trimming and whitespace collapse",
			input: "<p>  hello \t\n  world  \n </p>",
			want:  "<p>\n  hello world\n</p>\n",
		},
		{
			name:  "void elements have no closing line",
			input: `<div><input type="text" name="q"><br><hr><img src="test.png"></div>`,
			want:  "<div>\n  <input name=\"q\" type=\"text\">\n  <br>\n  <hr>\n  <img src=\"test.png\">\n</div>\n",
		},
		{
			name:  "tr fragment keeps its row and cells",
			input: `<tr><td>first</td><td>second</td></tr>`,
			want:  "<tr>\n  <td>\n    first\n  </td>\n  <td>\n    second\n  </td>\n</tr>\n",
		},
		{
			name:  "option fragment kept",
			input: `<option value="1">One</option><option value="2">Two</option>`,
			want:  "<option value=\"1\">\n  One\n</option>\n<option value=\"2\">\n  Two\n</option>\n",
		},
		{
			name:  "optgroup and option fragment",
			input: `<optgroup label="g"><option value="1">One</option></optgroup>`,
			want:  "<optgroup label=\"g\">\n  <option value=\"1\">\n    One\n  </option>\n</optgroup>\n",
		},
		{
			name:  "template content printed as children",
			input: `<template><div id="nested">Content</div></template>`,
			want:  "<template>\n  <div id=\"nested\">\n    Content\n  </div>\n</template>\n",
		},
		{
			name:  "non-void element with no children is one line",
			input: `<div></div>`,
			want:  "<div></div>\n",
		},
		{
			name:  "non-void element with only whitespace is one line",
			input: "<span id=\"empty\">   \n\t  </span>",
			want:  "<span id=\"empty\"></span>\n",
		},
		{
			name:  "comments are dropped",
			input: `<div><!-- this comment should be dropped --><span>Text</span></div>`,
			want:  "<div>\n  <span>\n    Text\n  </span>\n</div>\n",
		},
		{
			name:  "td and th fragment context",
			input: `<th>Header</th><td>Data</td>`,
			want:  "<th>\n  Header\n</th>\n<td>\n  Data\n</td>\n",
		},
		{
			name:  "col fragment context",
			input: `<col span="2">`,
			want:  "<col span=\"2\">\n",
		},
		{
			name:  "li fragment context",
			input: `<li>Item 1</li><li>Item 2</li>`,
			want:  "<li>\n  Item 1\n</li>\n<li>\n  Item 2\n</li>\n",
		},
		{
			name:  "dt and dd fragment context",
			input: `<dt>Term</dt><dd>Definition</dd>`,
			want:  "<dt>\n  Term\n</dt>\n<dd>\n  Definition\n</dd>\n",
		},
		{
			name:  "attribute escaping",
			input: `<a title="He said &quot;Hello&quot; &amp; smiled">Link</a>`,
			want:  "<a title=\"He said &#34;Hello&#34; &amp; smiled\">\n  Link\n</a>\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uitest.Normalise(tt.input)
			if got != tt.want {
				t.Errorf("Normalise() mismatch\ngot:\n%q\nwant:\n%q", got, tt.want)
			}
		})
	}
}

func TestNormaliseEscapesText(t *testing.T) {
	got := uitest.Normalise("<p>&lt;b&gt;bold&lt;/b&gt;</p>")
	if !strings.Contains(got, "&lt;b&gt;bold&lt;/b&gt;") || strings.Contains(got, "\n  <b>") {
		t.Errorf("text that looks like markup must stay escaped:\n%s", got)
	}
}
