//go:build mips && linux

// @generated
/*
Copyright (c) Facebook, Inc. and its affiliates.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package unix

type ifreq struct {
	Ifrn [16]byte
	Ifru [16]byte
}

const (
	PTP_CLOCK_GETCAPS   = 0x40503d01 //nolint:revive
	PTP_CLOCK_GETCAPS2  = 0x40503d0a //nolint:revive
	PTP_ENABLE_PPS      = 0x80043d04 //nolint:revive
	PTP_ENABLE_PPS2     = 0x80043d0d //nolint:revive
	PTP_EXTTS_REQUEST   = 0x80103d02 //nolint:revive
	PTP_EXTTS_REQUEST2  = 0x80103d0b //nolint:revive
	PTP_MASK_CLEAR_ALL  = 0x20003d13 //nolint:revive
	PTP_MASK_EN_SINGLE  = 0x80043d14 //nolint:revive
	PTP_PEROUT_REQUEST  = 0x80383d03 //nolint:revive
	PTP_PEROUT_REQUEST2 = 0x80383d0c //nolint:revive
	PTP_PIN_SETFUNC     = 0x80603d07 //nolint:revive
	PTP_PIN_SETFUNC2    = 0x80603d10 //nolint:revive
	PTP_SYS_OFFSET      = 0x83403d05 //nolint:revive
	PTP_SYS_OFFSET2     = 0x83403d0e //nolint:revive
)
