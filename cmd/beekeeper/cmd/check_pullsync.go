package cmd

import (
	"errors"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check/pullsync"
	"github.com/ethersphere/beekeeper/pkg/random"

	"github.com/spf13/cobra"
)

func (c *command) initCheckPullSync() *cobra.Command {
	const (
		optionNameUploadNodeCount   = "upload-node-count"
		optionNameReplicationFactor = "replication-factor"
		optionNameChunksPerNode     = "chunks-per-node"
		optionNameSeed              = "seed"
	)

	cmd := &cobra.Command{
		Use:   "pullsync",
		Short: "Checks pullsync ability of the cluster",
		Long:  `Checks pullsync ability of the cluster.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if c.config.GetInt(optionNameUploadNodeCount) > c.config.GetInt(optionNameNodeCount) {
				return errors.New("bad parameters: upload-node-count must be less or equal to node-count")
			}

			cluster := bee.NewCluster("bee", bee.ClusterOptions{
				APIDomain:           c.config.GetString(optionNameAPIDomain),
				APIInsecureTLS:      insecureTLSAPI,
				APIScheme:           c.config.GetString(optionNameAPIScheme),
				DebugAPIDomain:      c.config.GetString(optionNameDebugAPIDomain),
				DebugAPIInsecureTLS: insecureTLSDebugAPI,
				DebugAPIScheme:      c.config.GetString(optionNameDebugAPIScheme),
				Namespace:           c.config.GetString(optionNameNamespace),
				DisableNamespace:    disableNamespace,
			})

			ngOptions := newDefaultNodeGroupOptions()
			cluster.AddNodeGroup("nodes", *ngOptions)
			ng := cluster.NodeGroup("nodes")

			for i := 0; i < c.config.GetInt(optionNameNodeCount); i++ {
				if err := ng.AddNode(fmt.Sprintf("bee-%d", i), bee.NodeOptions{}); err != nil {
					return fmt.Errorf("adding node bee-%d: %s", i, err)
				}
			}

			var seed int64
			if cmd.Flags().Changed("seed") {
				seed = c.config.GetInt64(optionNameSeed)
			} else {
				seed = random.Int64()
			}

			return pullsync.Check(cluster, pullsync.Options{
				NodeGroup:                  "nodes",
				UploadNodeCount:            c.config.GetInt(optionNameUploadNodeCount),
				ReplicationFactorThreshold: c.config.GetInt(optionNameReplicationFactor),
				ChunksPerNode:              c.config.GetInt(optionNameChunksPerNode),
				Seed:                       seed,
			})
		},
		PreRunE: c.checkPreRunE,
	}

	cmd.Flags().IntP(optionNameUploadNodeCount, "u", 1, "number of nodes to upload chunks to")
	cmd.Flags().IntP(optionNameReplicationFactor, "r", 2, "minimal replication factor per chunk")
	cmd.Flags().IntP(optionNameChunksPerNode, "p", 1, "number of chunks to upload per node")
	cmd.Flags().Int64P(optionNameSeed, "s", 0, "seed for generating chunks; if not set, will be random")

	return cmd
}
