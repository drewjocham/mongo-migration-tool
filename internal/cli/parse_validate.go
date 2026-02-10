package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/drewjocham/mongo-migration-tool/internal/jsonutil"
	"github.com/drewjocham/mongo-migration-tool/internal/parser"
	"github.com/spf13/cobra"
)

func newParseCmd() *cobra.Command {
	var (
		input  string
		format string
	)

	cmd := &cobra.Command{
		Use:   "parse",
		Short: "Parse a payload (JSON or BSON) and normalize to JSON",
		RunE: func(cmd *cobra.Command, _ []string) error {
			raw, err := readPayload(cmd.InOrStdin(), input)
			if err != nil {
				return err
			}

			parsed, err := parser.ParseMap(raw, parser.WithFormat(parseFormat(format)))
			if err != nil {
				return err
			}

			enc := jsonutil.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(parsed)
		},
	}

	cmd.Flags().StringVarP(&input, "input", "i", "", "Input file (defaults to stdin)")
	cmd.Flags().StringVar(&format, "format", "json", "Input format: json or bson")
	return cmd
}

func newValidateCmd() *cobra.Command {
	var (
		input     string
		format    string
		typeField string
		typeName  string
	)

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Parse and validate a payload using registered types",
		RunE: func(cmd *cobra.Command, _ []string) error {
			raw, err := readPayload(cmd.InOrStdin(), input)
			if err != nil {
				return err
			}

			if typeName == "" && typeField == "" {
				return fmt.Errorf("provide --type or --type-field")
			}

			if typeName != "" {
				ctor := parser.DefaultRegistry[strings.ToLower(typeName)]
				if ctor == nil {
					return fmt.Errorf("no registered type: %s", typeName)
				}
				instance := ctor()
				if err := parser.ParseInto(raw, instance, parser.WithFormat(
					parseFormat(format)), parser.WithValidation(true)); err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), "valid")
				return nil
			}

			if _, err := parser.ParseByType(raw, typeField, parser.DefaultRegistry,
				parser.WithFormat(parseFormat(format)), parser.WithValidation(true)); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "valid")
			return nil
		},
	}

	cmd.Flags().StringVarP(&input, "input", "i", "", "Input file (defaults to stdin)")
	cmd.Flags().StringVar(&format, "format", "json", "Input format: json or bson")
	cmd.Flags().StringVar(&typeField, "type-field", "", "Discriminator field path (e.g. metadata.type)")
	cmd.Flags().StringVar(&typeName, "type", "", "Explicit type name to validate")
	return cmd
}

func readPayload(in io.Reader, path string) ([]byte, error) {
	if path == "" {
		return io.ReadAll(in)
	}
	return os.ReadFile(path)
}

func parseFormat(format string) parser.Format {
	switch strings.ToLower(format) {
	case "bson":
		return parser.FormatBSON
	default:
		return parser.FormatJSON
	}
}
