//go:build go1.12
// +build go1.12

package gmf

/*

#cgo pkg-config: libavcodec libavutil

#include <string.h>
#include <stdint.h>

#include "libavcodec/avcodec.h"
#include "libavutil/channel_layout.h"
#include "libavutil/samplefmt.h"
#include "libavutil/opt.h"
#include "libavutil/mem.h"
#include "libavutil/bprint.h"

#define HAVE_THREADS 1

static int32_t gmf_select_sample_fmt(AVCodec *codec)
{
    if (codec && codec->sample_fmts) {
        return codec->sample_fmts[0];
    }

    return -1;
}

static int check_sample_fmt(AVCodec *codec, enum AVSampleFormat sample_fmt) {
    const enum AVSampleFormat *p = codec->sample_fmts;

    while (*p != AV_SAMPLE_FMT_NONE) {
        if (*p == sample_fmt)
            return 1;
        p++;
    }
    return 0;
}

static int select_sample_rate(AVCodec *codec) {
    const int *p;
    int best_samplerate = 0;

    if (!codec->supported_samplerates)
        return 44100;

    p = codec->supported_samplerates;
    while (*p) {
        best_samplerate = FFMAX(*p, best_samplerate);
        p++;
    }
    return best_samplerate;
}

static int gmf_check_sample_rate(AVCodec *codec, int input_sample_rate) {
    const int *p;

    if ((p = codec->supported_samplerates) == NULL) {
        return 1;
    }

    while (*p) {
        if (*p == input_sample_rate) {
            return 1;
        }
        p++;
    }
    return 0;
}

static void call_av_freep(AVCodecContext *out){
    return av_freep(&out);
}

*/
import "C"

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"
)

const (
	AVCOL_RANGE_UNSPECIFIED = iota
	AVCOL_RANGE_MPEG        ///< the normal 219*2^(n-8) "MPEG" YUV ranges
	AVCOL_RANGE_JPEG        ///< the normal     2^n-1   "JPEG" YUV ranges
	AVCOL_RANGE_NB          ///< Not part of ABI
)

var (
	AV_CODEC_ID_MPEG1VIDEO int = C.AV_CODEC_ID_MPEG1VIDEO
	AV_CODEC_ID_MPEG2VIDEO int = C.AV_CODEC_ID_MPEG2VIDEO
	AV_CODEC_ID_H264       int = C.AV_CODEC_ID_H264
	AV_CODEC_ID_H265       int = C.AV_CODEC_ID_H265
	AV_CODEC_ID_MPEG4      int = C.AV_CODEC_ID_MPEG4
	AV_CODEC_ID_JPEG2000   int = C.AV_CODEC_ID_JPEG2000
	AV_CODEC_ID_MJPEG      int = C.AV_CODEC_ID_MJPEG
	AV_CODEC_ID_MSMPEG4V1  int = C.AV_CODEC_ID_MSMPEG4V1
	AV_CODEC_ID_MSMPEG4V2  int = C.AV_CODEC_ID_MSMPEG4V2
	AV_CODEC_ID_MSMPEG4V3  int = C.AV_CODEC_ID_MSMPEG4V3
	AV_CODEC_ID_WMV1       int = C.AV_CODEC_ID_WMV1
	AV_CODEC_ID_WMV2       int = C.AV_CODEC_ID_WMV2
	AV_CODEC_ID_FLV1       int = C.AV_CODEC_ID_FLV1
	AV_CODEC_ID_PNG        int = C.AV_CODEC_ID_PNG
	AV_CODEC_ID_TIFF       int = C.AV_CODEC_ID_TIFF
	AV_CODEC_ID_GIF        int = C.AV_CODEC_ID_GIF
	AV_CODEC_ID_RAWVIDEO   int = C.AV_CODEC_ID_RAWVIDEO

	AV_CODEC_ID_AAC int = C.AV_CODEC_ID_AAC

	CODEC_FLAG_GLOBAL_HEADER int   = C.AV_CODEC_FLAG_GLOBAL_HEADER
	AV_CODEC_FLAG_QSCALE     int32 = C.AV_CODEC_FLAG_QSCALE

	FF_MB_DECISION_SIMPLE int = C.FF_MB_DECISION_SIMPLE
	FF_MB_DECISION_BITS   int = C.FF_MB_DECISION_BITS
	FF_MB_DECISION_RD     int = C.FF_MB_DECISION_RD

	FF_QP2LAMBDA int = C.FF_QP2LAMBDA

	AV_SAMPLE_FMT_U8  int32 = C.AV_SAMPLE_FMT_U8
	AV_SAMPLE_FMT_S16 int32 = C.AV_SAMPLE_FMT_S16
	AV_SAMPLE_FMT_S32 int32 = C.AV_SAMPLE_FMT_S32
	AV_SAMPLE_FMT_FLT int32 = C.AV_SAMPLE_FMT_FLT
	AV_SAMPLE_FMT_DBL int32 = C.AV_SAMPLE_FMT_DBL

	AV_SAMPLE_FMT_U8P  int32 = C.AV_SAMPLE_FMT_U8P
	AV_SAMPLE_FMT_S16P int32 = C.AV_SAMPLE_FMT_S16P
	AV_SAMPLE_FMT_S32P int32 = C.AV_SAMPLE_FMT_S32P
	AV_SAMPLE_FMT_FLTP int32 = C.AV_SAMPLE_FMT_FLTP
	AV_SAMPLE_FMT_DBLP int32 = C.AV_SAMPLE_FMT_DBLP

	color_range_names map[uint32]string = map[uint32]string{
		AVCOL_RANGE_UNSPECIFIED: "unknown",
		AVCOL_RANGE_MPEG:        "tv",
		AVCOL_RANGE_JPEG:        "pc",
	}
)

type avBprint C.struct_AVBprint

type CodecCtx struct {
	codec      *Codec
	avCodecCtx *C.struct_AVCodecContext
	CgoMemoryManage
}

func NewCodecCtx(codec *Codec, options ...[]*Option) *CodecCtx {
	result := &CodecCtx{codec: codec}

	codecctx := C.avcodec_alloc_context3(codec.avCodec)
	if codecctx == nil {
		return nil
	}

	result.avCodecCtx = codecctx

	// we're really expecting only one options-array —
	// variadic arg is used for backward compatibility
	if len(options) == 1 {
		for _, option := range options[0] {
			option.Set(result.avCodecCtx)
		}
	}

	result.avCodecCtx.codec_id = codec.avCodec.id

	return result
}

func (cc *CodecCtx) SetOptions(options []Option) {
	for _, option := range options {
		option.Set(cc.avCodecCtx)
	}
}

/*
func (cc *CodecCtx) CopyExtra(ist *Stream) *CodecCtx {
	codec := cc.avCodecCtx
	icodec := ist.CodecCtx().avCodecCtx

	codec.bits_per_raw_sample = icodec.bits_per_raw_sample
	codec.chroma_sample_location = icodec.chroma_sample_location

	codec.codec_id = icodec.codec_id
	codec.codec_type = icodec.codec_type

	// codec.codec_tag = icodec.codec_tag

	codec.rc_max_rate = icodec.rc_max_rate
	codec.rc_buffer_size = icodec.rc_buffer_size

	codec.field_order = icodec.field_order

	codec.extradata = (*C.uint8_t)(C.av_mallocz((C.size_t)((C.uint64_t)(icodec.extradata_size) + C.AV_INPUT_BUFFER_PADDING_SIZE)))

	C.memcpy(unsafe.Pointer(codec.extradata), unsafe.Pointer(icodec.extradata), (C.size_t)(icodec.extradata_size))
	codec.extradata_size = icodec.extradata_size
	codec.bits_per_coded_sample = icodec.bits_per_coded_sample

	codec.has_b_frames = icodec.has_b_frames

	return cc
}*/

// SetExtradata TODO: Improving performance
//   Free or avcodec_free_context can free extradata
func (cc *CodecCtx) SetExtradata(extradata []byte) *CodecCtx {
	codec := cc.avCodecCtx
	codec.extradata_size = C.int(len(extradata))
	codec.extradata = (*C.uint8_t)(C.av_mallocz((C.size_t)((C.uint64_t)(len(extradata)) + C.AV_INPUT_BUFFER_PADDING_SIZE)))
	tmp := unsafe.Pointer(C.CBytes(extradata))
	C.memcpy(unsafe.Pointer(codec.extradata), tmp, (C.size_t)(codec.extradata_size))
	C.free(tmp)
	return cc
}

func (cc *CodecCtx) Open(dict *Dict) error {
	if cc.IsOpen() {
		return nil
	}

	var avDict *C.struct_AVDictionary
	if dict != nil {
		avDict = dict.avDict
	}

	if cc.Codec().IsDecoder() {
		cc.avCodecCtx.thread_count = 4
		cc.avCodecCtx.thread_type = 3
	}

	if averr := C.avcodec_open2(cc.avCodecCtx, cc.codec.avCodec, &avDict); averr < 0 {
		return errors.New(fmt.Sprintf("Error opening codec '%s:%s', averror: %s", cc.codec.Name(), cc.codec.LongName(), AvError(int(averr))))
	}

	return nil
}

// codec context is freed by avformat_free_context()
func (cc *CodecCtx) Free() {
	if cc.avCodecCtx != nil {
		C.avcodec_free_context(&cc.avCodecCtx)
	}
}

func (cc *CodecCtx) CloseAndRelease() {
	panic("(CodecCtx)CloseAndRelease() is deprecated")
}

func (cc *CodecCtx) Close() {
	C.avcodec_close(cc.avCodecCtx)
}

// @todo
func (cc *CodecCtx) SetOpt() {
	// mock
	C.av_opt_set_int(unsafe.Pointer(cc.avCodecCtx), C.CString("refcounted_frames"), 1, 0)
}

func (cc *CodecCtx) Codec() *Codec {
	return &Codec{avCodec: cc.avCodecCtx.codec}
}

func (cc *CodecCtx) Id() int {
	return int(cc.avCodecCtx.codec_id)
}

func (cc *CodecCtx) Type() int32 {
	return int32(cc.avCodecCtx.codec_type)
}

func (cc *CodecCtx) Width() int {
	return int(cc.avCodecCtx.width)
}

func (cc *CodecCtx) Height() int {
	return int(cc.avCodecCtx.height)
}

func (cc *CodecCtx) PixFmt() int32 {
	return int32(cc.avCodecCtx.pix_fmt)
}

func (cc *CodecCtx) FrameSize() int {
	return int(cc.avCodecCtx.frame_size)
}

func (cc *CodecCtx) SampleFmt() int32 {
	return cc.avCodecCtx.sample_fmt
}

func (cc *CodecCtx) SampleRate() int {
	return int(cc.avCodecCtx.sample_rate)
}

func (cc *CodecCtx) Profile() int {
	return int(cc.avCodecCtx.profile)
}

func (cc *CodecCtx) IsOpen() bool {
	return int(C.avcodec_is_open(cc.avCodecCtx)) > 0
}

func (cc *CodecCtx) SetProfile(profile int) *CodecCtx {
	cc.avCodecCtx.profile = C.int(profile)
	return cc
}

func (cc *CodecCtx) TimeBase() AVRational {
	return AVRational(cc.avCodecCtx.time_base)
}

func (cc *CodecCtx) ChannelLayout() int {
	return int(cc.avCodecCtx.channel_layout)
}
func (cc *CodecCtx) SetChannelLayout(channelLayout int) {
	cc.avCodecCtx.channel_layout = C.uint64_t(channelLayout)
}

func (cc *CodecCtx) BitRate() int {
	return int(cc.avCodecCtx.bit_rate)
}

func (cc *CodecCtx) Channels() int {
	return int(cc.avCodecCtx.channels)
}

func (cc *CodecCtx) SetBitRate(val int) *CodecCtx {
	cc.avCodecCtx.bit_rate = C.int64_t(val)
	return cc
}

func (cc *CodecCtx) SetWidth(val int) *CodecCtx {
	cc.avCodecCtx.width = C.int(val)
	return cc
}

func (cc *CodecCtx) SetHeight(val int) *CodecCtx {
	cc.avCodecCtx.height = C.int(val)
	return cc
}

func (cc *CodecCtx) SetDimension(w, h int) *CodecCtx {
	cc.avCodecCtx.width = C.int(w)
	cc.avCodecCtx.height = C.int(h)
	return cc
}

func (cc *CodecCtx) SetTimeBase(val AVR) *CodecCtx {
	cc.avCodecCtx.time_base.num = C.int(val.Num)
	cc.avCodecCtx.time_base.den = C.int(val.Den)
	return cc
}

func (cc *CodecCtx) SetGopSize(val int) *CodecCtx {
	cc.avCodecCtx.gop_size = C.int(val)
	return cc
}

func (cc *CodecCtx) GetGopSize() int {
	return int(cc.avCodecCtx.gop_size)
}

func (cc *CodecCtx) SetMaxBFrames(val int) *CodecCtx {
	cc.avCodecCtx.max_b_frames = C.int(val)
	return cc
}

func (cc *CodecCtx) SetPixFmt(val int32) *CodecCtx {
	cc.avCodecCtx.pix_fmt = val
	return cc
}

func (cc *CodecCtx) SetFlag(flag int) *CodecCtx {
	cc.avCodecCtx.flags |= C.int(flag)
	return cc
}

func (cc *CodecCtx) SetMbDecision(val int) *CodecCtx {
	cc.avCodecCtx.mb_decision = C.int(val)
	return cc
}

func (cc *CodecCtx) SetSampleFmt(val int32) *CodecCtx {
	if int(C.check_sample_fmt(cc.codec.avCodec, val)) == 0 {
		panic(fmt.Sprintf("encoder doesn't support sample format %s", GetSampleFmtName(val)))
	}

	cc.avCodecCtx.sample_fmt = val
	return cc
}

func (cc *CodecCtx) SetSampleRate(val int) *CodecCtx {
	cc.avCodecCtx.sample_rate = C.int(val)
	return cc
}

var (
	FF_COMPLIANCE_VERY_STRICT  int = C.FF_COMPLIANCE_VERY_STRICT
	FF_COMPLIANCE_STRICT       int = C.FF_COMPLIANCE_STRICT
	FF_COMPLIANCE_NORMAL       int = C.FF_COMPLIANCE_NORMAL
	FF_COMPLIANCE_UNOFFICIAL   int = C.FF_COMPLIANCE_UNOFFICIAL
	FF_COMPLIANCE_EXPERIMENTAL int = C.FF_COMPLIANCE_EXPERIMENTAL
)

func (cc *CodecCtx) SetStrictCompliance(val int) *CodecCtx {
	cc.avCodecCtx.strict_std_compliance = C.int(val)
	return cc
}

func (cc *CodecCtx) SetHasBframes(val int) *CodecCtx {
	cc.avCodecCtx.has_b_frames = C.int(val)
	return cc
}

func (cc *CodecCtx) SetChannels(val int) *CodecCtx {
	cc.avCodecCtx.channels = C.int(val)
	return cc
}

func (cc *CodecCtx) SetFrameRate(r AVR) *CodecCtx {
	cc.avCodecCtx.framerate.num = C.int(r.Num)
	cc.avCodecCtx.framerate.den = C.int(r.Den)
	return cc
}

func (cc *CodecCtx) SetBitsPerRawSample(val int) *CodecCtx {
	cc.avCodecCtx.bits_per_raw_sample = C.int(val)
	return cc
}

func (cc *CodecCtx) SelectSampleRate() int {
	return int(C.select_sample_rate(cc.codec.avCodec))
}

func (cc *CodecCtx) FlushBuffers() {
	C.avcodec_flush_buffers(cc.avCodecCtx)
}

func (cc *CodecCtx) Dump() {
	fmt.Println(cc.avCodecCtx)
}

func (cc *CodecCtx) GetFrameRate() AVRational {
	return AVRational(cc.avCodecCtx.framerate)
}

func (cc *CodecCtx) GetProfile() int {
	return int(cc.avCodecCtx.profile)
}

func (cc *CodecCtx) GetProfileName() string {
	return C.GoString(C.avcodec_profile_name(cc.avCodecCtx.codec_id, cc.avCodecCtx.profile))
}

func (cc *CodecCtx) GetMediaType() string {
	return C.GoString(C.av_get_media_type_string(cc.avCodecCtx.codec_type))
}

func (cc *CodecCtx) GetCodecTag() uint32 {
	return uint32(cc.avCodecCtx.codec_tag)
}

func (cc *CodecCtx) GetCodecTagName() string {
	var (
		ct     uint32 = uint32(cc.avCodecCtx.codec_tag)
		result string
	)

	for i := 0; i < 4; i++ {
		c := ct & 0xff
		result += fmt.Sprintf("%c", c)
		ct >>= 8
	}

	return fmt.Sprintf("%v", result)
}

func (cc *CodecCtx) GetCodedWith() int {
	return int(cc.avCodecCtx.coded_width)
}

func (cc *CodecCtx) GetCodedHeight() int {
	return int(cc.avCodecCtx.coded_height)
}

func (cc *CodecCtx) GetBFrames() int {
	return int(cc.avCodecCtx.has_b_frames)
}

func (cc *CodecCtx) GetPixFmtName() string {
	// return C.GoString(C.av_get_pix_fmt_name(cc.avCodecCtx.pix_fmt))
	return "unknown"
}

func (cc *CodecCtx) GetColorRangeName() string {
	return color_range_names[cc.avCodecCtx.color_range]
}

func (cc *CodecCtx) GetRefs() int {
	return int(cc.avCodecCtx.refs)
}

func (cc *CodecCtx) GetSampleFmtName() string {
	return C.GoString(C.av_get_sample_fmt_name(cc.avCodecCtx.sample_fmt))
}

/*
func (cc *CodecCtx) GetDefaultChannelLayout(ac int) int {
	return int(C.av_get_default_channel_layout(C.int(ac)))
}*/

func (cc *CodecCtx) GetBitsPerSample() int {
	return int(C.av_get_bits_per_sample(cc.codec.avCodec.id))
}

func (cc *CodecCtx) GetVideoSize() string {
	return fmt.Sprintf("%dx%d", cc.Width(), cc.Height())
}

func (cc *CodecCtx) GetAspectRation() AVRational {
	return AVRational(cc.avCodecCtx.sample_aspect_ratio)
}

func (cc *CodecCtx) Decode(pkt *Packet) ([]*Frame, error) {
	var (
		ret    int
		result []*Frame = make([]*Frame, 0)
	)

	if pkt == nil {
		ret = int(C.avcodec_send_packet(cc.avCodecCtx, nil))
	} else {
		ret = int(C.avcodec_send_packet(cc.avCodecCtx, &pkt.avPacket))
	}
	if ret < 0 {
		return nil, AvError(ret)
	}

	for {
		frame := NewFrame()

		ret = int(C.avcodec_receive_frame(cc.avCodecCtx, frame.avFrame))
		if AvErrno(ret) == syscall.EAGAIN || ret == AVERROR_EOF {
			frame.Free()
			break
		} else if ret < 0 {
			frame.Free()
			return nil, AvError(ret)
		}

		result = append(result, frame)
	}

	return result, nil
}

func (cc *CodecCtx) Encode(frames []*Frame, drain int) ([]*Packet, error) {
	var (
		ret    int
		result []*Packet = make([]*Packet, 0)
	)

	if len(frames) == 0 && drain >= 0 {
		frames = append(frames, nil)
	}

	for _, frame := range frames {
		if frame == nil {
			ret = int(C.avcodec_send_frame(cc.avCodecCtx, nil))
		} else {
			ret = int(C.avcodec_send_frame(cc.avCodecCtx, frame.avFrame))
		}
		if ret < 0 {
			return nil, AvError(ret)
		}

		for {
			pkt := NewPacket()
			ret = int(C.avcodec_receive_packet(cc.avCodecCtx, &pkt.avPacket))
			if ret < 0 {
				pkt.Free()
				break
			}

			result = append(result, pkt)
		}
		if frame != nil {
			frame.Free()
		}
	}

	return result, nil
}

func (cc *CodecCtx) Decode2(pkt *Packet) (*Frame, int) {
	var (
		ret int
	)

	if pkt == nil {
		ret = int(C.avcodec_send_packet(cc.avCodecCtx, nil))
	} else {
		ret = int(C.avcodec_send_packet(cc.avCodecCtx, &pkt.avPacket))
	}
	if ret < 0 {
		return nil, ret
	}

	frame := NewFrame()

	if ret = int(C.avcodec_receive_frame(cc.avCodecCtx, frame.avFrame)); ret < 0 {
		return nil, ret
	}

	return frame, 0
}

func (cc *CodecCtx) SelectSampleFmt() int32 {
	return int32(C.gmf_select_sample_fmt(cc.codec.avCodec))
}

func (cc *CodecCtx) SupportedSampleRate(val int) bool {
	return int(C.gmf_check_sample_rate(cc.codec.avCodec, C.int(val))) == 1
}

func (cc *CodecCtx) SetGlobalQuality(val int) {
	cc.avCodecCtx.global_quality = C.int(val)
}

func (cc *CodecCtx) SetPktTimeBase(val AVR) {
	cc.avCodecCtx.pkt_timebase.num = C.int(val.Num)
	cc.avCodecCtx.pkt_timebase.den = C.int(val.Den)
}

func (cc *CodecCtx) SetThreadCount(val int) {
	cc.avCodecCtx.thread_count = C.int(val)
}
