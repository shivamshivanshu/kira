package cli

import (
	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/schema"
)

func newSchemaCmd(_ *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "schema",
		Short: "Print the JSON Schema for kira's --json output shapes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			doc, err := schema.Generate()
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(doc)
			return err
		},
	}
}
