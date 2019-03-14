package cmds

import (
	"bytes"
	"fmt"
	"os"
	"text/tabwriter"
)

type TableWriter struct {
	Writer *tabwriter.Writer
	Header []string
	Values [][]string
}

func NewTableWriter(header []string, values [][]string) *TableWriter {
	return &TableWriter{
		Writer: tabwriter.NewWriter(os.Stdout, 10, 1, 3, ' ', 0),
		Header: header,
		Values: values,
	}
}

func (tw *TableWriter) Write() {
	fmt.Fprint(tw.Writer, stringListToTabDelimString(tw.Header))
	for _, value := range tw.Values {
		fmt.Fprint(tw.Writer, stringListToTabDelimString(value))
	}

	tw.Writer.Flush()
}

func stringListToTabDelimString(values []string) string {
	buffer := bytes.Buffer{}

	for _, v := range values {
		appendTabDelim(&buffer, v)
	}

	buffer.WriteString("\n")

	return buffer.String()
}

func appendTabDelim(buf *bytes.Buffer, value string) {
	if buf.Len() == 0 {
		buf.WriteString(value)
	} else {
		buf.WriteString("\t")
		buf.WriteString(value)
	}
}
