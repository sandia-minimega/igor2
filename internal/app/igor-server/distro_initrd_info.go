package igorserver

import (
	"bytes"
	"fmt"
	"github.com/sassoftware/go-rpmutils/cpio"
	"github.com/ulikunitz/xz"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"
)

// InitrdJob represents a pending task to process an initrd file.
type InitrdJob struct {
	Image *DistroImage // Unique identifier for the distro image.
}

// InitrdJobQueue implements a simple job queue for processing initrd files.
// A single worker will process jobs one at a time.
type InitrdJobQueue struct {
	queue chan InitrdJob
}

// NewInitrdJobQueue Resets any in-flight jobs that need to be
// re-queued due to a server shutdown and returns a new instance
// of the job queue.
func NewInitrdJobQueue() *InitrdJobQueue {

	err := performDbTx(func(tx *gorm.DB) error {
		// On startup:
		return tx.Model(&DistroImage{}).
			Where("initrd_info = ?", "pending").
			Update("initrd_info", "").Error
	})

	if err != nil {
		logger.Error().Msgf("Failed to revert pending InitrdInfo fields: %v", err)
	}

	return &InitrdJobQueue{
		queue: make(chan InitrdJob, 100),
	}
}

// Start launches the single worker that processes jobs sequentially.
func (q *InitrdJobQueue) Start() {
	go func() {
		for job := range q.queue {
			q.processInitrdJob(job)
		}
	}()
}

// Enqueue adds a new job to the queue.
func (q *InitrdJobQueue) Enqueue(job InitrdJob) {
	q.queue <- job
}

// EnqueuePendingJobs should be called at startup. It scans the database
// for any DistroImage records with InitrdInfo either empty or "pending"
// and enqueues them in batches to handle a large backlog gracefully.
func (q *InitrdJobQueue) EnqueuePendingJobs() {
	const batchSize = 25

	for {
		var pendingImages []DistroImage
		var err error

		// We now only look for images with initrd_info = "" (truly unprocessed).
		var emptyInfoQuery = map[string]interface{}{"initrd_info": ""}

		err = performDbTx(func(tx *gorm.DB) error {
			pendingImages, err = dbReadImage(emptyInfoQuery, batchSize, tx)
			if err != nil {
				return err
			}
			// If no more images to process, just return (the loop will break).
			if len(pendingImages) == 0 {
				return nil
			}

			ids := make([]string, 0, len(pendingImages))
			for _, img := range pendingImages {
				ids = append(ids, img.ImageID)
			}

			if pendingErr := tx.Model(&DistroImage{}).
				Where("image_id IN ?", ids).
				Update("initrd_info", "pending").Error; pendingErr != nil {
				return pendingErr
			}

			return nil
		})

		if err != nil {
			logger.Error().Msgf("Failed to retrieve or mark pending distro images: %v", err)
			return
		}

		// If no images were found in this iteration, we're done.
		if len(pendingImages) == 0 {
			break
		}

		logger.Info().Msgf("Found %d images needing initrd_info updates", len(pendingImages))

		for i := range pendingImages {
			logger.Info().Msgf("Enqueueing initrd job for image %s (now marked 'pending')", pendingImages[i].ImageID)
			q.Enqueue(InitrdJob{Image: &pendingImages[i]})
		}

		// Optional: Sleep briefly between batches to avoid hammering the DB in a tight loop.
		time.Sleep(100 * time.Millisecond)
	}
}

// processInitrdJob processes an individual initrd job.
func (q *InitrdJobQueue) processInitrdJob(job InitrdJob) {

	var image = job.Image
	logger.Debug().Msgf("Processing initrd job for image %s", image.ImageID)

	// Check if the image is still pending (or empty).
	if image.InitrdInfo != "" {
		logger.Debug().Msgf("Skipping image %s; InitrdInfo is already set to '%s'", image.ImageID, image.InitrdInfo)
		return
	}

	imageInitrdPath := filepath.Join(getImageStorePath(image.ImageID), image.Initrd)

	// Process the initrd file using the existing method.
	// parseInitrdInfo opens the file, decompresses it, and searches for OS release info.
	initrdInfo := parseInitrdInfo(imageInitrdPath)

	// If no useful information is found, update to "unknown" and trigger notification.
	if initrdInfo == "not found" || initrdInfo == "unknown" {
		initrdInfo = "unknown"
		notifyAdministrators()
	}

	// Update the DistroImage record with the initrd info.
	image.InitrdInfo = initrdInfo

	dbAccess.Lock()

	saveErr := performDbTx(func(tx *gorm.DB) error {
		return tx.Model(&DistroImage{}).
			Where("image_id = ?", job.Image.ImageID).
			Update("initrd_info", initrdInfo).Error
	})

	dbAccess.Unlock()

	if saveErr != nil {
		logger.Error().Msgf("Failed to update DistroImage %s with initrd info: %v", image.ImageID, saveErr)
		return
	}

	logger.Debug().Msgf("Updated image %s with initrd info: %s", image.ImageID, initrdInfo)
}

// notifyAdministrators is a placeholder function for actions needed when
// an initrd file has been processed but no OS release information was found.
// For example, this might trigger an email to administrators.
func notifyAdministrators() {
	// Add your notification logic here.
	logger.Warn().Msg("notifyAdministrators: initrd info not found; administrators should be notified")
}

func parseInitrdInfo(iPath string) (iInfo string) {

	iInfo = "not found"

	file, err := os.Open(iPath)
	if err != nil {
		logger.Error().Msgf("error opening file %s: %v", iPath, err)
		return
	}
	defer file.Close()

	// Try to create xz reader
	xzReader, err := xz.NewReader(file)
	if err == nil {
		// Stream the xz decompression into the cpio reader
		iInfo = processCpioArchive(xzReader)
	} else {
		logger.Warn().Msgf("unable to create XZ compression reader: %v", err)
	}

	logger.Debug().Msgf("os version: %s", iInfo)
	return
}

func processCpioArchive(reader io.Reader) string {
	// Read the entire cpio archive into memory
	var buf bytes.Buffer
	_, err := io.Copy(&buf, reader)
	if err != nil {
		logger.Warn().Msgf("error reading cpio archive into memory: %v", err)
		return "unknown"
	}
	data := buf.Bytes()

	// Define the list of files to check for OS version information
	primaryFilesToCheck := []string{
		"etc/os-release",
		"usr/lib/initrd-release",
	}
	secondaryFilesToCheck := []string{
		"etc/lsb-release",
		"etc/redhat-release",
		"etc/alpine-release",
		"etc/debian_version",
		"etc/fedora-release",
		"etc/centos-release",
		"etc/SuSE-release",
		"etc/arch-release",
		"etc/slackware-version",
		"etc/gentoo-release",
	}

	// First pass: Check primary files
	for _, fileToCheck := range primaryFilesToCheck {
		if info := checkFileInCpioArchive(data, fileToCheck); info != "" {
			return info
		}
	}

	// Second pass: Check secondary files
	for _, fileToCheck := range secondaryFilesToCheck {
		if info := checkFileInCpioArchive(data, fileToCheck); info != "" {
			return info
		}
	}

	return "unknown"
}

func checkFileInCpioArchive(data []byte, fileToCheck string) string {
	cpioReader := cpio.NewReader(bytes.NewReader(data))
	for {
		hdr, err := cpioReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.Warn().Msgf("error reading cpio archive: %v", err)
			return "unknown"
		}

		filename := strings.TrimPrefix(hdr.Filename(), "/")
		if filename == fileToCheck {
			var content strings.Builder
			_, err := io.Copy(&content, cpioReader)
			if err != nil {
				logger.Warn().Msgf("error reading file content: %v", err)
				return "unknown"
			}

			// Parse the content based on the filename
			switch filename {
			case "etc/os-release":
				return parseOSRelease(content.String())
			case "usr/lib/initrd-release":
				return parseInitrdRelease(content.String())
			case "etc/lsb-release":
				return parseLSBRelease(content.String())
			case "etc/redhat-release", "etc/centos-release", "etc/fedora-release":
				return parseRedhatRelease(content.String())
			case "etc/alpine-release":
				return parseAlpineRelease(content.String())
			case "etc/debian_version":
				return parseDebianVersion(content.String())
			case "etc/SuSE-release":
				return parseSuSERelease(content.String())
			case "etc/arch-release":
				return parseArchRelease(content.String())
			case "etc/slackware-version":
				return parseSlackwareVersion(content.String())
			case "etc/gentoo-release":
				return parseGentooRelease(content.String())
			}
		}
	}
	return ""
}

func parseOSRelease(content string) string {
	logger.Debug().Msgf("Found /etc/os-release file. Scanning...")

	if !strings.Contains(content, "NAME=") {
		return ""
	}

	lines := strings.Split(content, "\n")
	var name, version string

	for _, line := range lines {
		if strings.HasPrefix(line, "NAME=") {
			name = extractValue(line)
		} else if strings.HasPrefix(line, "VERSION_ID=") {
			version = extractValue(line)
		}
	}

	if name == "Red Hat Enterprise Linux" {
		name = "RHEL"
	}

	output := fmt.Sprintf("%s %s", name, version)
	return strings.TrimSpace(output)
}

func parseInitrdRelease(content string) string {
	logger.Debug().Msgf("Found /usr/lib/initrd-release file. Scanning...")

	if !strings.Contains(content, "NAME=") {
		return ""
	}

	lines := strings.Split(content, "\n")

	var name, version, releaseNickname string

	for _, line := range lines {
		if strings.HasPrefix(line, "NAME=") {
			name = extractValue(line)
		} else if strings.HasPrefix(line, "VERSION=") {
			version = extractVersion(line)
			releaseNickname = extractReleaseNickname(line)
		}
	}

	if name == "Red Hat Enterprise Linux" {
		name = "RHEL"
	}

	output := fmt.Sprintf("%s %s %s", name, version, releaseNickname)

	return strings.TrimSpace(output)
}

func parseLSBRelease(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "DISTRIB_DESCRIPTION=") {
			return strings.TrimSpace(strings.ReplaceAll(strings.TrimPrefix(line, "DISTRIB_DESCRIPTION="), "\"", ""))
		}
	}

	return "unknown (/etc/lsb-release)"
}

func parseRedhatRelease(content string) string {
	return strings.TrimSpace(content)
}

func parseAlpineRelease(content string) string {
	return fmt.Sprintf("Alpine Linux %s", strings.TrimSpace(content))
}

func parseDebianVersion(content string) string {
	return fmt.Sprintf("Debian %s", strings.TrimSpace(content))
}

func parseSuSERelease(content string) string {
	return strings.TrimSpace(content)
}

func parseArchRelease(content string) string {
	return fmt.Sprintf("Arch Linux %s", strings.TrimSpace(content))
}

func parseSlackwareVersion(content string) string {
	return fmt.Sprintf("Slackware %s", strings.TrimSpace(content))
}

func parseGentooRelease(content string) string {
	return fmt.Sprintf("Gentoo %s", strings.TrimSpace(content))
}

// Helper function to extract the value from a key-value pair
func extractValue(line string) string {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) == 2 {
		return strings.Trim(parts[1], `"`)
	}
	return ""
}

// Helper function to extract the version from the VERSION line
func extractVersion(line string) string {
	value := extractValue(line)
	parts := strings.Split(value, " ")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func extractReleaseNickname(line string) string {
	value := extractValue(line)
	start := strings.Index(value, "(")
	end := strings.Index(value, ")")
	if start != -1 && end != -1 && end > start {
		return value[start : end+1]
	}
	return ""
}
