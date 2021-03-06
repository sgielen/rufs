(
	cat <<EOF
package web

var staticFiles = map[string]string{
EOF

	mkdir -p dist
	find dist -type f | while read filename; do
		cat <<EOF
  "${filename#dist}": \`$(cat $filename | sed -e 's/`/` + "`" + `/g')\`,
EOF
	done

	cat <<EOF
}
EOF
) | gofmt
