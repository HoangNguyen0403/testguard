package demo

import com.google.common.truth.Truth.assertThat
import io.mockk.mockk
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Test

class CoroutineAdvanceNegativeTest {
    private val dispatcher = StandardTestDispatcher()

    @Test
    fun advancedStateFlow() = runTest(dispatcher) {
        val vm = SearchViewModel(repo = mockk(relaxed = true), dispatcher = dispatcher)
        vm.search("kotlin")
        advanceUntilIdle()
        assertThat(vm.state.value.loading).isFalse()
    }
}
