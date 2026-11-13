package demo

import com.google.common.truth.Truth.assertThat
import io.mockk.mockk
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.runTest
import org.junit.Test

class CoroutineAdvancePositiveTest {
    private val dispatcher = StandardTestDispatcher()

    @Test
    fun unadvancedStateFlow() = runTest(dispatcher) {
        val vm = SearchViewModel(repo = mockk(relaxed = true), dispatcher = dispatcher)
        vm.search("kotlin")
        assertThat(vm.state.value.loading).isFalse()
    }
}
