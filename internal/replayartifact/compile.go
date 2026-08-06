package replayartifact

import (
	"context"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/massive"
	"github.com/belandresj/live-equities-momentum-scanner/internal/reference"
)

const (
	temporaryPrefix = ".aggregate-replay.tmp-"
	leaseName       = ".aggregate-replay.compile.lock"
)

type CompileState string

const (
	CompileComplete             CompileState = "complete"
	CompileFailed               CompileState = "failed"
	CompileCanceled             CompileState = "canceled"
	CompilePersistenceUncertain CompileState = "persistence_uncertain"
)

type CompileReason string

const (
	CompileReasonNone                CompileReason = ""
	CompileReasonPlanBudget          CompileReason = "plan_budget"
	CompileReasonDownload            CompileReason = "download"
	CompileReasonCoverage            CompileReason = "coverage"
	CompileReasonSchemaCanonical     CompileReason = "schema_canonical_encoding"
	CompileReasonSummaryDigest       CompileReason = "summary_digest_truncation"
	CompileReasonExistingDestination CompileReason = "existing_destination"
	CompileReasonPersistence         CompileReason = "persistence"
	CompileReasonCanceled            CompileReason = "canceled"
)

type Limits struct {
	MaximumNormalizedRecords int64
	MaximumResponseBytes     int64
	MaximumArtifactBytes     int64
	MaximumTemporaryBytes    int64
	MaximumTemporaryFiles    int
	MaximumInMemoryRecords   int64
}

type CompletePlan struct {
	Binding              reference.Binding
	Start, End           time.Time
	Workers              int
	DestinationDirectory string
	Limits               Limits
}

type PartialPlan struct {
	Input                 PartialInput
	DestinationDirectory  string
	MaximumTemporaryBytes int64
	MaximumTemporaryFiles int
}

type CompileAccounting struct {
	Complete             int64
	Failed               int64
	Canceled             int64
	PersistenceUncertain int64
}

type CompileResult struct {
	State      CompileState
	Reason     CompileReason
	ArtifactID string
	Path       string
	Download   massive.DownloadAccounting
	Outcomes   []massive.SymbolOutcome
	Accounting CompileAccounting
}

type persistenceOps struct {
	link          func(string, string) error
	remove        func(string) error
	syncDirectory func(string) error
	beforeLink    func()
}

var ordinaryPersistence = persistenceOps{
	link:   os.Link,
	remove: os.Remove,
	syncDirectory: func(path string) error {
		directory, err := os.Open(path)
		if err != nil {
			return err
		}
		defer directory.Close()
		return directory.Sync()
	},
}

func Compile(ctx context.Context, plan CompletePlan, downloader *massive.OfflineDownloader) CompileResult {
	return compileWithOps(ctx, plan, downloader, ordinaryPersistence)
}

func compileWithOps(ctx context.Context, plan CompletePlan, downloader *massive.OfflineDownloader, ops persistenceOps) CompileResult {
	result := CompileResult{State: CompileFailed, Reason: CompileReasonPlanBudget}
	if ctx == nil || downloader == nil || !validCompletePlan(plan) || ops.link == nil || ops.remove == nil || ops.syncDirectory == nil {
		return finishCompile(result)
	}
	operation, reason := beginOperation(plan.DestinationDirectory, plan.Limits.MaximumTemporaryFiles)
	if reason != CompileReasonNone {
		result.Reason = reason
		return finishCompile(result)
	}
	releaseBeforePublication := func() bool {
		return operation.release(ops.remove) == nil
	}

	download := downloader.Download(ctx, massive.DownloadPlan{Binding: plan.Binding, Start: plan.Start, End: plan.End, Workers: plan.Workers,
		MaximumNormalizedRecords: plan.Limits.MaximumNormalizedRecords, MaximumResponseBytes: plan.Limits.MaximumResponseBytes})
	result.Download = download.Accounting()
	result.Outcomes = download.Outcomes()
	if !download.Complete() {
		if !releaseBeforePublication() {
			result.Reason = CompileReasonPersistence
		} else if ctx.Err() != nil || download.Accounting().CanceledSymbols > 0 {
			result.State, result.Reason = CompileCanceled, CompileReasonCanceled
		} else {
			result.Reason = CompileReasonDownload
		}
		return finishCompile(result)
	}
	if download.Accounting().Records > plan.Limits.MaximumInMemoryRecords {
		result.Reason = CompileReasonPlanBudget
		if !releaseBeforePublication() {
			result.Reason = CompileReasonPersistence
		}
		return finishCompile(result)
	}
	artifact, metadata, err := buildComplete(plan.Binding, plan.Start, plan.End, download, plan.Limits.MaximumArtifactBytes, plan.Limits.MaximumNormalizedRecords)
	if err != nil || int64(len(artifact)) > plan.Limits.MaximumTemporaryBytes {
		result.Reason = CompileReasonSchemaCanonical
		if !releaseBeforePublication() {
			result.Reason = CompileReasonPersistence
		}
		return finishCompile(result)
	}
	publication := publish(ctx, operation, plan.DestinationDirectory, artifact, metadata,
		ValidationPlan{plan.Binding, plan.Start, plan.End, CompleteFinalBars, plan.Limits.MaximumArtifactBytes, plan.Limits.MaximumNormalizedRecords}, ops)
	result.State, result.Reason, result.ArtifactID, result.Path = publication.state, publication.reason, publication.artifactID, publication.path
	return finishCompile(result)
}

func CompilePartialArtifact(ctx context.Context, plan PartialPlan) CompileResult {
	return compilePartialWithOps(ctx, plan, ordinaryPersistence)
}

func compilePartialWithOps(ctx context.Context, plan PartialPlan, ops persistenceOps) CompileResult {
	result := CompileResult{State: CompileFailed, Reason: CompileReasonPlanBudget}
	if ctx == nil || plan.DestinationDirectory == "" || plan.MaximumTemporaryBytes <= 0 || !validTemporaryFileLimit(plan.MaximumTemporaryFiles) ||
		plan.Input.MaximumBytes <= 0 || plan.Input.MaximumBytes > plan.MaximumTemporaryBytes || ops.link == nil || ops.remove == nil || ops.syncDirectory == nil {
		return finishCompile(result)
	}
	operation, reason := beginOperation(plan.DestinationDirectory, plan.MaximumTemporaryFiles)
	if reason != CompileReasonNone {
		result.Reason = reason
		return finishCompile(result)
	}
	artifact, metadata, err := BuildPartial(plan.Input)
	if err != nil || int64(len(artifact)) > plan.MaximumTemporaryBytes {
		result.Reason = CompileReasonSchemaCanonical
		if operation.release(ops.remove) != nil {
			result.Reason = CompileReasonPersistence
		}
		return finishCompile(result)
	}
	publication := publish(ctx, operation, plan.DestinationDirectory, artifact, metadata,
		ValidationPlan{plan.Input.Binding, plan.Input.Start, plan.Input.End, PartialSynthetic, plan.Input.MaximumBytes, plan.Input.MaximumRecords}, ops)
	result.State, result.Reason, result.ArtifactID, result.Path = publication.state, publication.reason, publication.artifactID, publication.path
	return finishCompile(result)
}

type publicationResult struct {
	state      CompileState
	reason     CompileReason
	artifactID string
	path       string
}

func publish(ctx context.Context, operation *operationFiles, directory string, artifact []byte, metadata Metadata, validation ValidationPlan, ops persistenceOps) publicationResult {
	result := publicationResult{state: CompileFailed, reason: CompileReasonPersistence}
	releaseBeforePublication := func() bool { return operation.release(ops.remove) == nil }
	temporary, err := os.CreateTemp(directory, temporaryPrefix)
	if err != nil {
		_ = releaseBeforePublication()
		return result
	}
	operation.temporary = temporary.Name()
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		_ = releaseBeforePublication()
		return result
	}
	if writeSyncClose(temporary, artifact) != nil {
		_ = releaseBeforePublication()
		return result
	}
	handle, err := OpenValidated(operation.temporary, validation)
	if err != nil || handle.Metadata().ArtifactID != metadata.ArtifactID {
		if handle != nil {
			handle.Close()
		}
		result.reason = CompileReasonSummaryDigest
		_ = releaseBeforePublication()
		return result
	}
	_ = handle.Close()
	finalPath := filepath.Join(directory, "aggregate-replay-"+strings.TrimPrefix(metadata.ArtifactID, "sha256:")+".jsonl")
	if ops.beforeLink != nil {
		ops.beforeLink()
	}
	if ctx.Err() != nil {
		result.state, result.reason = CompileCanceled, CompileReasonCanceled
		if !releaseBeforePublication() {
			result.state, result.reason = CompileFailed, CompileReasonPersistence
		}
		return result
	}
	linkErr := ops.link(operation.temporary, finalPath)
	if linkErr != nil {
		if errors.Is(linkErr, os.ErrExist) || errors.Is(linkErr, syscall.EEXIST) {
			existing, validationErr := OpenValidated(finalPath, validation)
			validExisting := validationErr == nil && existing.Metadata().ArtifactID == metadata.ArtifactID
			if existing != nil {
				existing.Close()
			}
			if validExisting {
				if !releaseBeforePublication() {
					return result
				}
				return publicationResult{CompileComplete, CompileReasonNone, metadata.ArtifactID, finalPath}
			}
			result.reason = CompileReasonExistingDestination
		}
		_ = releaseBeforePublication()
		return result
	}

	cleanupFailed := false
	if err := ops.remove(operation.temporary); err != nil {
		cleanupFailed = true
	} else {
		operation.temporary = ""
	}
	if err := ops.remove(operation.lease); err != nil {
		cleanupFailed = true
	} else {
		operation.lease = ""
	}
	syncErr := ops.syncDirectory(directory)
	if cleanupFailed || syncErr != nil {
		return publicationResult{CompilePersistenceUncertain, CompileReasonPersistence, metadata.ArtifactID, ""}
	}
	return publicationResult{CompileComplete, CompileReasonNone, metadata.ArtifactID, finalPath}
}

func validCompletePlan(plan CompletePlan) bool {
	limits := plan.Limits
	if plan.DestinationDirectory == "" || plan.Workers < 1 || plan.Workers > massive.OfflineWorkerLimit ||
		!validReplayInterval(plan.Binding, plan.Start, plan.End) || limits.MaximumNormalizedRecords <= 0 || limits.MaximumResponseBytes <= 0 ||
		limits.MaximumArtifactBytes <= 0 || limits.MaximumTemporaryBytes <= 0 || !validTemporaryFileLimit(limits.MaximumTemporaryFiles) || limits.MaximumInMemoryRecords <= 0 ||
		limits.MaximumInMemoryRecords > limits.MaximumNormalizedRecords || limits.MaximumArtifactBytes > limits.MaximumTemporaryBytes {
		return false
	}
	seconds := int64(plan.End.Sub(plan.Start) / time.Second)
	symbols := int64(len(plan.Binding.UniverseSymbols()))
	return seconds > 0 && symbols > 0 && symbols <= math.MaxInt64/seconds && symbols*seconds <= limits.MaximumInMemoryRecords
}

type operationFiles struct {
	lease     string
	temporary string
}

func beginOperation(directory string, maximumTemporaryFiles int) (*operationFiles, CompileReason) {
	if !validTemporaryFileLimit(maximumTemporaryFiles) {
		return nil, CompileReasonPlanBudget
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, CompileReasonPersistence
	}
	info, err := os.Lstat(directory)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, CompileReasonPersistence
	}
	lease := filepath.Join(directory, leaseName)
	file, err := os.OpenFile(lease, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, CompileReasonPersistence
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(lease)
		return nil, CompileReasonPersistence
	}
	operation := &operationFiles{lease: lease}
	directoryFile, err := os.Open(directory)
	if err != nil {
		_ = operation.release(os.Remove)
		return nil, CompileReasonPersistence
	}
	names, readErr := directoryFile.Readdirnames(maximumTemporaryFiles + 2)
	directoryFile.Close()
	if readErr != nil && !errors.Is(readErr, io.EOF) || len(names) > maximumTemporaryFiles+1 {
		_ = operation.release(os.Remove)
		return nil, CompileReasonPlanBudget
	}
	for _, name := range names {
		if strings.HasPrefix(name, temporaryPrefix) {
			_ = operation.release(os.Remove)
			return nil, CompileReasonPersistence
		}
	}
	return operation, CompileReasonNone
}

func validTemporaryFileLimit(value int) bool {
	return value > 0 && value <= int(^uint(0)>>1)-2
}

func (o *operationFiles) release(remove func(string) error) error {
	var retained error
	if o.temporary != "" {
		if err := remove(o.temporary); err != nil && !errors.Is(err, os.ErrNotExist) {
			retained = err
		} else {
			o.temporary = ""
		}
	}
	if o.lease != "" {
		if err := remove(o.lease); err != nil && !errors.Is(err, os.ErrNotExist) {
			retained = err
		} else {
			o.lease = ""
		}
	}
	return retained
}

func writeSyncClose(file *os.File, value []byte) error {
	if _, err := file.Write(value); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

func finishCompile(result CompileResult) CompileResult {
	result.Accounting = CompileAccounting{}
	switch result.State {
	case CompileComplete:
		result.Accounting.Complete = 1
	case CompileCanceled:
		result.Accounting.Canceled = 1
	case CompilePersistenceUncertain:
		result.Accounting.PersistenceUncertain = 1
	default:
		result.State = CompileFailed
		result.Accounting.Failed = 1
	}
	return result
}
