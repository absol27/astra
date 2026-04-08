package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	graph "github.com/TSELab/astra/internal/graph"
	"github.com/TSELab/astra/internal/mapper"
	parser "github.com/TSELab/astra/internal/parser"
	buildinfoparser "github.com/TSELab/astra/internal/parser/buildinfo"
	gitparser "github.com/TSELab/astra/internal/parser/git"
	intotoparser "github.com/TSELab/astra/internal/parser/intoto"
	slsaparser "github.com/TSELab/astra/internal/parser/slsa"
)

func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

var parseCmd = &cobra.Command{
	Use:   "parse",
	Short: "Parse a provenance source into normalized records",
	RunE: func(cmd *cobra.Command, args []string) error {
		in, _ := cmd.Flags().GetString("input")
		out, _ := cmd.Flags().GetString("output")
		format, _ := cmd.Flags().GetString("format")

		var p parser.Parser
		var r io.Reader

		switch format {
		case "git":
			p = &gitparser.GitParser{}
			r = strings.NewReader(in)
		case "buildinfo":
			p = &buildinfoparser.BuildinfoParser{}
			f, err := os.Open(in)
			if err != nil {
				return err
			}
			defer f.Close()
			r = f
		case "intoto":
			p = &intotoparser.InTotoParser{}
			f, err := os.Open(in)
			if err != nil {
				return err
			}
			defer f.Close()
			r = f
		case "slsa":
			p = &slsaparser.SlsaParser{}
			f, err := os.Open(in)
			if err != nil {
				return err
			}
			defer f.Close()
			r = f
		default:
			return fmt.Errorf("unknown format: %s", format)
		}

		data, err := p.Parse(r)
		if err != nil {
			return err
		}
		if err := writeJSON(out, data); err != nil {
			return err
		}
		fmt.Println("[OK] Parsed ->", out)
		return nil
	},
}

var mapCmd = &cobra.Command{
	Use:   "map",
	Short: "Map normalized records to an AStRA graph",
	RunE: func(cmd *cobra.Command, args []string) error {
		in, _ := cmd.Flags().GetString("input")
		out, _ := cmd.Flags().GetString("output")

		b, err := os.ReadFile(in)
		if err != nil {
			return err
		}
		var parsed parser.Mapped
		if err := json.Unmarshal(b, &parsed); err != nil {
			return err
		}

		astra := mapper.ToAstraGraph(parsed)
		if err := writeJSON(out, astra); err != nil {
			return err
		}
		fmt.Println("[OK] Mapped ->", out)
		return nil
	},
}

var vizCmd = &cobra.Command{
	Use:   "viz",
	Short: "Render an AStRA graph as DOT",
	RunE: func(cmd *cobra.Command, args []string) error {
		in, _ := cmd.Flags().GetString("input")
		out, _ := cmd.Flags().GetString("output")

		graphJSON, err := os.ReadFile(in)
		if err != nil {
			return err
		}
		var g graph.AstraGraph
		if err := json.Unmarshal(graphJSON, &g); err != nil {
			return err
		}
		dot := graph.ToDOT(g)
		if err := os.WriteFile(out, []byte(dot), 0o644); err != nil {
			return err
		}
		fmt.Println("[OK] DOT graph written to", out)
		return nil
	},
}

var rootCmd = &cobra.Command{
	Use:   "astra",
	Short: "AStRA provenance graph tool",
}

func init() {
	parseCmd.Flags().StringP("input", "i", "", "input file path or repo URL (git)")
	parseCmd.Flags().StringP("output", "o", "", "output normalized JSON")
	parseCmd.Flags().StringP("format", "f", "git", "format: git|buildinfo|intoto|slsa")
	parseCmd.MarkFlagRequired("input")
	parseCmd.MarkFlagRequired("output")

	mapCmd.Flags().StringP("input", "i", "", "input parsed JSON (parser.Mapped)")
	mapCmd.Flags().StringP("output", "o", "", "output AStRA graph JSON")
	mapCmd.MarkFlagRequired("input")
	mapCmd.MarkFlagRequired("output")

	vizCmd.Flags().StringP("input", "i", "", "input graph JSON")
	vizCmd.Flags().StringP("output", "o", "graph.dot", "output DOT file")
	vizCmd.MarkFlagRequired("input")

	rootCmd.AddCommand(parseCmd, mapCmd, vizCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
