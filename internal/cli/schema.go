package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/schema"
)

func newSchemaCmd(g *globalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Print the JSON Schema for kira's --json output shapes and automation payload",
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
	cmd.AddCommand(newSchemaInitCmd(g))
	return cmd
}

func newSchemaInitCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Write the built-in entity schemas a repo is missing, leaving existing files alone",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			written, err := s.MaterializeSchemas(cfg)
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), datamodel.SchemaInitResult{Written: written})
			}
			if len(written) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Every built-in schema is already present.")
				return nil
			}
			for _, p := range written {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Wrote "+p)
			}
			return nil
		},
	}
}
