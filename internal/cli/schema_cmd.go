package cli

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/drewjocham/mongo-migration-tool/internal/jsonutil"
	"github.com/drewjocham/mongo-migration-tool/internal/schema"
	"github.com/spf13/cobra"
)

func newSchemaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "schema",
		Short:       "Schema utilities",
		Annotations: map[string]string{annotationOffline: "true"},
	}

	cmd.AddCommand(newSchemaIndexesCmd())
	return cmd
}

func newSchemaIndexesCmd() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:         "indexes",
		Short:       "List expected indexes registered in code",
		Annotations: map[string]string{annotationOffline: "true"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			switch strings.ToLower(output) {
			case "json":
				return renderIndexesJSON(cmd.OutOrStdout())
			case "table", "":
				renderIndexesTable(cmd.OutOrStdout())
				return nil
			default:
				return fmt.Errorf("unsupported output format: %s", output)
			}
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "table", "Output format: table or json")
	return cmd
}

func renderIndexesJSON(w io.Writer) error {
	encoder := jsonutil.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(schema.Indexes())
}

func renderIndexesTable(w io.Writer) {
	indexes := schema.Indexes()
	if len(indexes) == 0 {
		fmt.Fprintln(w, "No index specifications registered.")
		return
	}

	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	fmt.Fprintln(tw, "COLLECTION\tINDEX\tKEYS\tUNIQUE\tSPARSE\tPARTIAL FILTER")
	fmt.Fprintln(tw, "----------\t-----\t----\t------\t------\t--------------")

	for _, spec := range indexes {
		unique := "no"
		if spec.Unique {
			unique = "yes"
		}
		sparse := "no"
		if spec.Sparse {
			sparse = "yes"
		}
		partial := spec.PartialFilterString()
		if partial == "" {
			partial = "-"
		}

		fmt.Fprintf(
			tw,
			"%s\t%s\t%s\t%s\t%s\t%s\n",
			spec.Collection,
			spec.Name,
			spec.KeyString(),
			unique,
			sparse,
			partial,
		)
	}

	tw.Flush()
}
