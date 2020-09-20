package algorithm

func QuickSort(array []int, start int, end int) []int {
	pivot := array[start]
	i, j := start, end
	for i < j {
		for i < j && array[j] >= pivot {
			j--
		}
		if i < j {
			array[i], array[j] = array[j], array[i]
		}
		for i < j && array[i] <= pivot {
			i++
		}
		if i < j {
			array[i], array[j] = array[j], array[i]
		}
	}
	array[i] = pivot
	if start < i-1 {
		QuickSort(array, start, i-1)
	}
	if i+1 < end {
		QuickSort(array, i+1, end)
	}
	return array
}
