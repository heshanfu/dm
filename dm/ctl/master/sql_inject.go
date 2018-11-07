package master

import (
	"context"
	"fmt"
	"strings"

	"github.com/juju/errors"
	"github.com/spf13/cobra"

	"github.com/pingcap/tidb-enterprise-tools/dm/ctl/common"
	"github.com/pingcap/tidb-enterprise-tools/dm/pb"
)

// NewSQLInjectCmd creates a SQLInject command
func NewSQLInjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sql-inject <-w worker> <task_name> <sql1;sql2;>",
		Short: "sql-inject injects (limited) sqls into syncer as binlog event",
		Run:   sqlInjectFunc,
	}
	return cmd
}

// sqlInjectFunc does sql inject request
func sqlInjectFunc(cmd *cobra.Command, _ []string) {
	if len(cmd.Flags().Args()) < 2 {
		fmt.Println(cmd.Usage())
		return
	}

	workers, err := common.GetWorkerArgs(cmd)
	if err != nil {
		common.PrintLines("%s", errors.ErrorStack(err))
		return
	}
	if len(workers) != 1 {
		common.PrintLines("want only one worker, but got %v", workers)
		return
	}

	taskName := cmd.Flags().Arg(0)
	if strings.TrimSpace(taskName) == "" {
		common.PrintLines("task_name is empty")
		return
	}

	extraArgs := cmd.Flags().Args()[1:]
	realSQLs, err := common.ExtractSQLsFromArgs(extraArgs)
	if err != nil {
		common.PrintLines("check sqls err %s", errors.ErrorStack(err))
		return
	}
	for _, sql := range realSQLs {
		isDDL, err2 := common.IsDDL(sql)
		if err2 != nil {
			common.PrintLines("check sql err %s", errors.ErrorStack(err2))
			return
		}
		if !isDDL {
			common.PrintLines("only support inject DDL currently, but got '%s'", sql)
			return
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cli := common.MasterClient()
	resp, err := cli.HandleSQLs(ctx, &pb.HandleSQLsRequest{
		Name:   taskName,
		Op:     pb.SQLOp_INJECT,
		Args:   realSQLs,
		Worker: workers[0],
	})
	if err != nil {
		common.PrintLines("can not inject sql:\n%v", errors.ErrorStack(err))
		return
	}

	common.PrettyPrintResponse(resp)
}
