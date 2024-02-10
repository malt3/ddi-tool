package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/malt3/ddi-tool/api/repart"
	"github.com/malt3/ddi-tool/pkg/ddi"
	"github.com/spf13/cobra"
)

var (
	repartJSON string
	blocksize  int
	ukiPath    string
)

func init() {
	finalizeCmd.Flags().StringVarP(&repartJSON, "repart-json", "r", "", "path systemd-repart json output")
	finalizeCmd.MarkFlagRequired("repart-json")
	finalizeCmd.Flags().IntVarP(&blocksize, "blocksize", "b", 0, "blocksize of the image")
	finalizeCmd.Flags().StringVarP(&ukiPath, "uki-path", "u", "", "path to the uki binary inside the EFI partition")
	rootCmd.AddCommand(finalizeCmd)
}

var finalizeCmd = &cobra.Command{
	Use:   "finalize [image]",
	Short: "Finalize a ddi built with systemd-repart",
	Long:  `After building a ddi with systemd-repart, this command can be used to finalize the image by injecting dm-verity hashes.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repartFile, err := os.ReadFile(repartJSON)
		if err != nil {
			return err
		}
		var repartJSON repart.Output
		if err := json.Unmarshal(repartFile, &repartJSON); err != nil {
			return err
		}
		var roothash, usrhash string
		for _, partition := range repartJSON {
			if len(partition.Roothash) > 0 {
				roothash = partition.Roothash
			}
			if len(partition.Usrhash) > 0 {
				usrhash = partition.Usrhash
			}
		}
		image, err := ddi.New(args[0], int64(blocksize), ukiPath)
		if err != nil {
			return err
		}
		defer image.Close()
		cmdline, err := image.GetCmdline()
		if err != nil {
			return err
		}
		if len(roothash) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "setting roothash=%s\n", roothash)
			if err := cmdline.SetOne("roothash", roothash, true); err != nil {
				return fmt.Errorf("setting roothash: %w", err)
			}
		}
		if len(usrhash) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "setting usrhash=%s\n", usrhash)
			if err := cmdline.SetOne("usrhash", usrhash, true); err != nil {
				return fmt.Errorf("setting usrhash: %w", err)
			}
		}
		after, err := cmdline.String()
		if err != nil {
			return err
		}
		fmt.Println(after)
		return nil
	},
}
