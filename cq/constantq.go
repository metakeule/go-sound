package cq

import (
  "fmt"
  "math"

  "github.com/mjibson/go-dsp/fft"

)

const DEBUG_CQ = true

type ConstantQ struct {
  kernel CQKernel

	octaves int
  bigBlockSize int
  outputLatency int
  buffers [][]float64

  latencies []int
  decimators []Resampler

}


func NewConstantQ(params CQParams) ConstantQ {
  octaves := int(math.Ceil(math.Log(params.maxFrequency / params.minFrequency) / math.Log(2))); // TODO: math.Log2
  if octaves < 1 {
    panic("Need at least one octave")
  }


  kernel := NewCQKernel(params)
  p := kernel.Properties

  // Use exact powers of two for resampling rates. They don't have
  // to be related to our actual samplerate: the resampler only
  // cares about the ratio, but it only accepts integer source and
  // target rates, and if we start from the actual samplerate we
  // risk getting non-integer rates for lower octaves
  sourceRate := unsafeShift(octaves)
  latencies := make([]int, octaves, octaves)
  decimators := make([]Resampler, octaves, octaves)

  // top octave, no resampling
  latencies[0] = 0
  decimators[0] = Resampler{} // HACK

  for i := 1; i < octaves; i++ {
    factor := unsafeShift(i)

    r := NewResampler(sourceRate, sourceRate / factor, 50, 0.05)
    if DEBUG_CQ {
      fmt.Printf("Forward: octave %d: resample from %v to %v\n", i, sourceRate, sourceRate / factor)
    }

    // We need to adapt the latencies so as to get the first input
    // sample to be aligned, in time, at the decimator output
    // across all octaves.
    //
    // Our decimator uses a linear phase filter, but being causal
    // it is not zero phase: it has a latency that depends on the
    // decimation factor. Those latencies have been calculated
    // per-octave and are available to us in the latencies
    // array. Left to its own devices, the first input sample will
    // appear at output sample 0 in the highest octave (where no
    // decimation is needed), sample number latencies[1] in the
    // next octave down, latencies[2] in the next one, etc. We get
    // to apply some artificial per-octave latency after the
    // decimator in the processing chain, in order to compensate
    // for the differing latencies associated with different
    // decimation factors. How much should we insert?
    //
    // The outputs of the decimators are at different rates (in
    // terms of the relation between clock time and samples) and
    // we want them aligned in terms of time. So, for example, a
    // latency of 10 samples with a decimation factor of 2 is
    // equivalent to a latency of 20 with no decimation -- they
    // both result in the first output sample happening at the
    // same equivalent time in milliseconds.
    //
    // So here we record the latency added by the decimator, in
    // terms of the sample rate of the undecimated signal. Then we
    // use that to compensate in a moment, when we've discovered
    // what the longest latency across all octaves is.
    latencies[i] = r.GetLatency() * factor
    decimators[i] = r
  }

  bigBlockSize := p.fftSize * unsafeShift(octaves - 1)

  // Now add in the extra padding and compensate for hops that must
  // be dropped in order to align the atom centres across
  // octaves. Again this is a bit trickier because we are doing it
  // at input rather than output and so must work in per-octave
  // sample rates rather than output blocks

  emptyHops := p.firstCentre / p.atomSpacing;

  drops := make([]int, octaves, octaves)
  for i := 0; i < octaves; i++ {
    factor := unsafeShift(i)
    dropHops := emptyHops * unsafeShift(octaves - i - 1) - emptyHops
    drops[i] = ((dropHops * p.fftHop) * factor) / p.atomsPerFrame
  }

  maxLatPlusDrop := 0
  for i := 0; i < octaves; i++ {
    latPlusDrop := latencies[i] + drops[i]
    if latPlusDrop > maxLatPlusDrop {
      maxLatPlusDrop = latPlusDrop
    }
  }

  totalLatency := maxLatPlusDrop
  
  lat0 := totalLatency - latencies[0] - drops[0]
  totalLatency = int(math.Ceil(float64((lat0 / p.fftHop) * p.fftHop))) + latencies[0] + drops[0]

  // We want (totalLatency - latencies[i]) to be a multiple of 2^i
  // for each octave i, so that we do not end up with fractional
  // octave latencies below. In theory this is hard, in practice if
  // we ensure it for the last octave we should be OK.
  finalOctLat := float64(latencies[octaves - 1])
  finalOneFactInt := unsafeShift(octaves - 1)
  finalOctFact := float64(finalOneFactInt)

  totalLatency = int(finalOctLat + finalOctFact * math.Ceil((float64(totalLatency) - finalOctLat) / finalOctFact) + .5)

  if DEBUG_CQ {
    fmt.Printf("total latency = %v\n", totalLatency)
  }

  // Padding as in the reference (will be introduced with the
  // latency compensation in the loop below)
  outputLatency := totalLatency + bigBlockSize - p.firstCentre * unsafeShift(octaves - 1)

  if DEBUG_CQ {
    fmt.Printf("bigBlockSize = %v, firstCentre = %v, octaves = %v, so outputLatency = %v\n",
        bigBlockSize, p.firstCentre, octaves, outputLatency)
  }


  buffers := make([][]float64, octaves, octaves)

  for i := 0; i < octaves; i++ {
    factor := unsafeShift(i)

    // Calculate the difference between the total latency applied
    // across all octaves, and the existing latency due to the
    // decimator for this octave, and then convert it back into
    // the sample rate appropriate for the output latency of this
    // decimator -- including one additional big block of padding
    // (as in the reference).

    octaveLatency := float64(totalLatency - latencies[i] - drops[i] + bigBlockSize) / float64(factor)

    if DEBUG_CQ {
      rounded := float64(round(octaveLatency))
      fmt.Printf("octave %d: resampler latency = %v, drop = %v, (/factor = %v), octaveLatency = %v -> %v (diff * factor = %v * %v = %v)\n",
          i, latencies[i], drops[i], drops[i] / factor, octaveLatency, rounded, octaveLatency - rounded, factor, (octaveLatency - rounded) * float64(factor))

      fmt.Printf("num(%v - %v - %v + %v) / %v = %v\n",
        totalLatency, latencies[i], drops[i], bigBlockSize, factor, octaveLatency)

    }

    sz := int(octaveLatency + 0.5)
    buffers[i] = make([]float64, sz, sz)
  }

  // m_fft = new FFTReal(m_p.fftSize);
  return ConstantQ {
    // params,
    kernel,

    octaves,
    bigBlockSize,
    outputLatency,
    buffers,

    latencies,
    decimators,
  }
}

func round(v float64) int {
  return int(v + 0.5)
}

func (cq ConstantQ) process(td []float64) [][]complex128 {
  for _, v := range td {
    // TODO - faster array append in golang?
    cq.buffers[0] = append(cq.buffers[0], v)
  }

  for i := 1; i < cq.octaves; i++ {
    dec := cq.decimators[i].Process(td)
    for _, v := range dec {
      cq.buffers[i] = append(cq.buffers[i], v)
    }
  }

  out := [][]complex128{}


  for {
    // We could have quite different remaining sample counts in
    // different octaves, because (apart from the predictable
    // added counts for decimator output on each block) we also
    // have variable additional latency per octave
    enough := true
    for i := 0; i < cq.octaves; i++ {
      required := cq.kernel.Properties.fftSize * unsafeShift(cq.octaves - i - 1)
      if len(cq.buffers[i]) < required {
        enough = false
      }
    }
    if !enough {
      break
    }

    base := len(out)
    totalColumns := unsafeShift(cq.octaves - 1) * cq.kernel.Properties.atomsPerFrame

    for i := 0; i < totalColumns; i++ {
      out = append(out, []complex128{})
    }

    for octave := 0; octave < cq.octaves; octave++ {
      blocksThisOctave := unsafeShift(cq.octaves - octave - 1)

      for b := 0; b < blocksThisOctave; b++ {
        block := cq.processOctaveBlock(octave)

        for j := 0; j < cq.kernel.Properties.atomsPerFrame; j++ {
          target := base + (b * (totalColumns / blocksThisOctave) + (j * ((totalColumns / blocksThisOctave) / cq.kernel.Properties.atomsPerFrame)))

          for len(out[target]) < cq.kernel.Properties.binsPerOctave * (octave + 1) {
            out[target] = append(out[target], complex(0, 0))
          }
          for i := 0; i < cq.kernel.Properties.binsPerOctave; i++ {
            out[target][cq.kernel.Properties.binsPerOctave * octave + i] = block[j][cq.kernel.Properties.binsPerOctave - i - 1]
          }
        }
      }
    }
  }

  return out
}

func (cq ConstantQ) getRemainingOutput() [][]complex128 {
  // Same as padding added at start, though rounded up
  pad := int(math.Ceil(float64(cq.outputLatency) / float64(cq.bigBlockSize))) * cq.bigBlockSize
  zeros := make([]float64, pad, pad)
  return cq.process(zeros);
}

func (cq ConstantQ) processOctaveBlock(octave int) [][]complex128 {
  // TODO - merge real pairs into complex array
  // ro, io := make([]float64, cq.kernel.Properties.fftSize, cq.kernel.Properties.fftSize), make([]float64, cq.kernel.Properties.fftSize, cq.kernel.Properties.fftSize)

  // HACK
  cv := fft.FFTReal(cq.buffers[octave])
  // cv := m_fft->forward(m_buffers[octave].data(), ro.data(), io.data());

  lshift := len(cq.buffers[octave]) - cq.kernel.Properties.fftHop
  shifted := make([]float64, lshift, lshift)
  for i := 0; i < lshift; i++ {
    shifted[i] = cq.buffers[octave][i + cq.kernel.Properties.fftHop]
  }
  cq.buffers[octave] = shifted

  // ComplexSequence cv;
  // for (int i = 0; i < m_p.fftSize; ++i) {
    // cv.push_back(Complex(ro[i], io[i]));
  // }
  cqrowvec := cq.kernel.processForward(cv)

  // Reform into a column matrix
  cqblock := make([][]complex128, cq.kernel.Properties.atomsPerFrame, cq.kernel.Properties.atomsPerFrame)
  for j := 0; j < cq.kernel.Properties.atomsPerFrame; j++ {
    cqblock[j] = make([]complex128, cq.kernel.Properties.binsPerOctave, cq.kernel.Properties.binsPerOctave)
    for i := 0; i < cq.kernel.Properties.binsPerOctave; i++ {
      cqblock[j][i] = cqrowvec[i * cq.kernel.Properties.atomsPerFrame + j]
    }
  }

  return cqblock;
}

func unsafeShift(s int) int {
  return 1 << uint(s)
}
