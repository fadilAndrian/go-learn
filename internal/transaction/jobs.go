package transaction

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

// sweepAge = pending baru dicek ulang setelah selama ini (beri waktu webhook datang dulu).
const sweepAge = time.Minute

type CheckPendingArgs struct {
	TransactionID int64 `json:"transaction_id"`
	MerchantID    int64 `json:"merchant_id"`
}

func (CheckPendingArgs) Kind() string { return "check_pending" }

// CheckPendingWorker = Service.Check; idempoten (guard from=pending), error → retry backoff River.
type CheckPendingWorker struct {
	river.WorkerDefaults[CheckPendingArgs]
	svc *Service
}

func (w *CheckPendingWorker) Work(ctx context.Context, job *river.Job[CheckPendingArgs]) error {
	_, err := w.svc.Check(ctx, job.Args.MerchantID, job.Args.TransactionID)
	return err
}

type SweepArgs struct{}

func (SweepArgs) Kind() string { return "sweep_pending" }

// SweepWorker enqueue CheckPending untuk tiap transaksi pending (unik per transaksi, tidak dobel).
// ponytail: batas 500 per sweep, sisanya ikut sweep berikutnya.
type SweepWorker struct {
	river.WorkerDefaults[SweepArgs]
	svc *Service
}

func (w *SweepWorker) Work(ctx context.Context, _ *river.Job[SweepArgs]) error {
	list, err := w.svc.repository.ListStalePending(ctx, sweepAge, 500)
	if err != nil || len(list) == 0 {
		return err
	}
	params := make([]river.InsertManyParams, len(list))
	for i, p := range list {
		params[i] = river.InsertManyParams{
			Args:       CheckPendingArgs{TransactionID: p.ID, MerchantID: p.MerchantID},
			InsertOpts: &river.InsertOpts{UniqueOpts: river.UniqueOpts{ByArgs: true}},
		}
	}
	_, err = river.ClientFromContext[pgx.Tx](ctx).InsertMany(ctx, params)
	return err
}

// AddWorkers mendaftarkan worker transaksi ke registry River milik main.
func AddWorkers(w *river.Workers, svc *Service) {
	river.AddWorker(w, &CheckPendingWorker{svc: svc})
	river.AddWorker(w, &SweepWorker{svc: svc})
}

// SweepEvery = jadwal sweep untuk river.Config.PeriodicJobs.
func SweepEvery(d time.Duration) *river.PeriodicJob {
	return river.NewPeriodicJob(river.PeriodicInterval(d),
		func() (river.JobArgs, *river.InsertOpts) { return SweepArgs{}, nil },
		&river.PeriodicJobOpts{RunOnStart: true})
}
