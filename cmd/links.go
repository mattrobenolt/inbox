package cmd

import (
	"fmt"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"go.withmatt.com/inbox/internal/config"
	"go.withmatt.com/inbox/internal/links"
)

var linksCmd = &cobra.Command{
	Use:   "links",
	Short: "Manage learned link redirect domains",
}

var linksListCmd = &cobra.Command{
	Use:   "list",
	Short: "List learned redirect domains",
	RunE:  runLinksList,
}

var linksSaveCmd = &cobra.Command{
	Use:   "save",
	Short: "Save learned redirect domains into config.toml",
	RunE:  runLinksSave,
}

func init() {
	linksCmd.AddCommand(linksListCmd)
	linksCmd.AddCommand(linksSaveCmd)
	rootCmd.AddCommand(linksCmd)
}

func runLinksList(cmd *cobra.Command, args []string) error {
	cache, err := links.LoadCache()
	if err != nil {
		return fmt.Errorf("unable to load link cache: %w", err)
	}

	entries, err := cache.ListDomainEntries()
	if err != nil {
		return fmt.Errorf("unable to list learned domains: %w", err)
	}
	if len(entries) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No learned domains.")
		return nil
	}

	writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "domain\tredirects\tlast_seen")
	fmt.Fprintln(writer, "------\t---------\t---------")
	for _, entry := range entries {
		lastSeen := ""
		if !entry.LastSeen.IsZero() {
			lastSeen = entry.LastSeen.Format(time.RFC3339)
		}
		fmt.Fprintf(writer, "%s\t%d\t%s\n", entry.Domain, entry.RedirectCount, lastSeen)
	}
	return writer.Flush()
}

func runLinksSave(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("unable to load config: %w", err)
	}

	cache, err := links.LoadCache()
	if err != nil {
		return fmt.Errorf("unable to load link cache: %w", err)
	}

	learned, err := cache.ListDomains()
	if err != nil {
		return fmt.Errorf("unable to list learned domains: %w", err)
	}
	if len(learned) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No learned domains to save.")
		return nil
	}

	existing := make(map[string]struct{}, len(cfg.Links.UnwrapDomains))
	for _, domain := range cfg.Links.UnwrapDomains {
		normalized := links.NormalizeDomain(domain)
		if normalized == "" {
			continue
		}
		existing[normalized] = struct{}{}
	}
	deny := make(map[string]struct{}, len(cfg.Links.DoNotResolve))
	for _, domain := range cfg.Links.DoNotResolve {
		normalized := links.NormalizeDomain(domain)
		if normalized == "" {
			continue
		}
		deny[normalized] = struct{}{}
	}

	sort.Strings(learned)
	added := make([]string, 0, len(learned))
	for _, domain := range learned {
		normalized := links.NormalizeDomain(domain)
		if normalized == "" {
			continue
		}
		if _, ok := deny[normalized]; ok {
			continue
		}
		if _, ok := existing[normalized]; ok {
			continue
		}
		existing[normalized] = struct{}{}
		cfg.Links.UnwrapDomains = append(cfg.Links.UnwrapDomains, normalized)
		added = append(added, normalized)
	}

	if len(added) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No new domains to save.")
		return nil
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("unable to save config: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Saved %d domains to config.\n", len(added))
	return nil
}
