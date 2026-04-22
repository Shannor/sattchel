package configs

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func get() *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "Get a config value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			value := viper.Get(key)
			if value == nil {
				fmt.Printf("no value for key: %s\n", key)
				return nil
			}
			fmt.Printf("key: %s, Value:%v\n", key, value)
			return nil
		},
	}
}
