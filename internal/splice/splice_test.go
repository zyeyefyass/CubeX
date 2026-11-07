package splice

import (
	"bytes"
	"testing"
)

func TestSpliceManager(t *testing.T) {
	mgr := NewManager()
	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}

	st := mgr.GetStatus()
	if !st.Enabled {
		t.Errorf("expected enabled=true, got %v", st.Enabled)
	}
	if st.PipeCapacityBytes != 65536 {
		t.Errorf("expected pipe capacity 65536, got %d", st.PipeCapacityBytes)
	}

	// Record splice
	mgr.RecordSplice(1024*1024, 20)
	st2 := mgr.GetStatus()
	if st2.TotalBytesSpliced <= st.TotalBytesSpliced {
		t.Errorf("expected TotalBytesSpliced to increase")
	}

	// Test Toggle
	toggled := mgr.Toggle(false)
	if toggled.Enabled {
		t.Errorf("expected toggle(false) to disable")
	}

	// Test Flush
	flushRes := mgr.FlushPipes()
	if !flushRes.Success {
		t.Errorf("expected flush success")
	}
	if mgr.GetStatus().SpliceActiveFlows != 0 {
		t.Errorf("expected 0 active flows after flush")
	}

	// Test SpliceCopy with buffers
	srcBuf := bytes.NewBuffer([]byte("HELLO CUBEX ZERO-COPY DATA PLANE"))
	dstBuf := bytes.NewBuffer(nil)

	written, err := mgr.SpliceCopy(dstBuf, srcBuf)
	if err != nil {
		t.Fatalf("SpliceCopy failed: %v", err)
	}
	if written != 32 {
		t.Errorf("expected 32 bytes written, got %d", written)
	}
	if dstBuf.String() != "HELLO CUBEX ZERO-COPY DATA PLANE" {
		t.Errorf("unexpected content: %s", dstBuf.String())
	}
}
