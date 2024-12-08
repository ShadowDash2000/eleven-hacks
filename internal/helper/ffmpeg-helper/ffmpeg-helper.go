package ffmpeghelper

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	ffmpeg "github.com/u2takey/ffmpeg-go"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type Probe struct {
	Streams []Stream `json:"streams"`
	Format  Format   `json:"format"`
}

type Format struct {
	Bitrate string `json:"bit_rate"`
}

type Stream struct {
	CodecType  string `json:"codec_type"`
	DurationTs int    `json:"duration_ts"`
	Duration   string `json:"duration"`
	RFrameRate string `json:"r_frame_rate"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	Bitrate    string `json:"bit_rate"`
}

func IsFfmpegAvailable() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

func IsFfprobeAvailable() bool {
	_, err := exec.LookPath("ffprobe")
	return err == nil
}

func SplitVideo(ctx context.Context, filePath, outputPath string, duration int) error {
	if duration <= 0 {
		return errors.New("duration must be a positive integer")
	}

	err := os.MkdirAll(outputPath, os.ModePerm)
	if err != nil {
		return err
	}

	videoDuration, err := GetVideoDuration(filePath)
	if err != nil {
		return err
	}

	audioTrackCount, err := GetAudioTrackCount(filePath)
	if err != nil {
		return err
	}

	fileName := filepath.Base(filePath)

	i := 1
	for startTime := 0; startTime < videoDuration; startTime += duration {

		outputFile := filepath.Join(outputPath, fmt.Sprintf("%d_%s", i, fileName))
		input := ffmpeg.Input(filePath, ffmpeg.KwArgs{"ss": startTime, "t": duration})
		video := input.Video().Filter("scale", ffmpeg.Args{"640", "-2"})
		audio := input.Audio()

		if audioTrackCount > 1 {
			audio = audio.
				Filter("amerge", ffmpeg.Args{fmt.Sprintf("inputs=%d", audioTrackCount)})
		}

		output := ffmpeg.Output(
			[]*ffmpeg.Stream{video, audio},
			outputFile,
			ffmpeg.KwArgs{
				"c:v":    "libx264",
				"preset": "ultrafast",
				"b:v":    "1500k",
				"c:a":    "aac",
				"map":    "0:v:0",
			},
		)

		output.Context = ctx
		err = output.Run()
		if err != nil {
			return err
		}

		i++
	}

	return nil
}

func GetVideoDuration(filePath string) (int, error) {
	probe := &Probe{}
	fileInfoJson, err := ffmpeg.Probe(filePath)
	if err != nil {
		return 0, err
	}

	err = json.Unmarshal([]byte(fileInfoJson), &probe)
	if err != nil {
		return 0, err
	}

	var duration string
	for _, stream := range probe.Streams {
		if stream.CodecType == "video" {
			duration = strings.Split(stream.Duration, ".")[0]
			break
		}
	}

	if duration == "" {
		return 0, fmt.Errorf("ffmpeg: duration is empty")
	}

	return strconv.Atoi(duration)
}

func GetVideoTrackCount(filePath string) (int, error) {
	return GetTrackCount(filePath, "video")
}

func GetAudioTrackCount(filePath string) (int, error) {
	return GetTrackCount(filePath, "audio")
}

func GetTrackCount(filePath, codecType string) (int, error) {
	probe := &Probe{}
	fileInfoJson, err := ffmpeg.Probe(filePath)
	if err != nil {
		return 0, err
	}

	err = json.Unmarshal([]byte(fileInfoJson), &probe)
	if err != nil {
		return 0, err
	}

	tracksCount := 0
	for _, stream := range probe.Streams {
		if stream.CodecType == codecType {
			tracksCount += 1
		}
	}

	return tracksCount, nil
}
