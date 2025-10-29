package main

import (
	"errors"
	"strconv"
	"strings"
)

type Packet struct {
	Time         string
	Latitude     float64
	Longitude    float64
	Satellites   int
	Acceleration [3]float64
}

func ParsePacket(line string) (Packet, error) {
	var p Packet

	line = strings.TrimSpace(line)
	if line == "" {
		return p, errors.New("empty line")
	}

	parts := strings.Split(line, ";")
	if len(parts) < 6 {
		return p, errors.New("not enough fields")
	}

	for _, field := range parts[1:] {
		switch {
		case strings.HasPrefix(field, "Time-"):
			p.Time = strings.TrimPrefix(field, "Time-")

		case strings.HasPrefix(field, "Latitude-"):
			v := strings.TrimPrefix(field, "Latitude-")
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return p, err
			}
			p.Latitude = f

		case strings.HasPrefix(field, "Longitude-"):
			v := strings.TrimPrefix(field, "Longitude-")
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return p, err
			}
			p.Longitude = f

		case strings.HasPrefix(field, "Satellites-"):
			v := strings.TrimPrefix(field, "Satellites-")
			n, err := strconv.Atoi(v)
			if err != nil {
				return p, err
			}
			p.Satellites = n

		case strings.HasPrefix(field, "Acceleration"):

			partsAcc := strings.Split(field, ":")
			if len(partsAcc) != 2 {
				return p, errors.New("bad accel field")
			}
			nums := strings.Split(partsAcc[1], ",")
			if len(nums) != 3 {
				return p, errors.New("bad accel values")
			}
			for i := 0; i < 3; i++ {
				f, err := strconv.ParseFloat(nums[i], 64)
				if err != nil {
					return p, err
				}
				p.Acceleration[i] = f
			}
		}
	}

	return p, nil
}
