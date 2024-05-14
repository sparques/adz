package adz

import (
	"fmt"
	"os"
	"testing"
)

func Test_Tailcall(t *testing.T) {
	interp := NewInterp()
	interp.Stdout = os.Stdout
	out, err := interp.ExecString(`
		proc fib n {
			if {eq $n 1} {return [int 1]}
			if {eq $n 0} {return [int 0]}
			
			return [+ [fib [+ $n -1]] [fib [+ $n -2]]]
		}
		if false {
		    
    if(n<0) return -1;
    if(n==1||n==0) return prev2;
    return fibonacci_tail_Recusive(n-1,prev2+prev1,prev1);
		}

		proc fibtc {n n1 n2} {
			if {or [== $n 1] [== $n1 0]} {return $n2}
			tailcall [+ $n [int -1]] [+ $n1 $n2] $n1
		}

		fibtc 50 1 1
		#fib 50
	`)

	fmt.Println(out.String, err)
}
