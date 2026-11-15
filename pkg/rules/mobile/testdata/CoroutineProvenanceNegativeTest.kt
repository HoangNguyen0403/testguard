package demo

import com.google.common.truth.Truth.assertThat
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.runTest
import org.junit.Test
import kotlin.test.assertEquals

class CoroutineProvenanceNegativeTest {
    private val dispatcher = StandardTestDispatcher()

    @Test
    fun subjectWithoutDispatcherInjection() = runTest(dispatcher) {
        val vm = SearchViewModel()
        vm.search("kotlin")
        assertThat(vm.state.value.loading).isFalse()
    }

    @Test
    fun subjectNeverCalled() = runTest(dispatcher) {
        val vm = SearchViewModel(dispatcher = dispatcher)
        assertThat(vm.state.value.loading).isFalse()
    }

    @Test
    fun assertionWithoutStateRead() = runTest(dispatcher) {
        val vm = SearchViewModel(dispatcher = dispatcher)
        vm.search("kotlin")
        assertThat(vm).isNotNull()
    }

    @Test
    fun unknownDispatcherProvenance() = runTest {
        val vm = SearchViewModel()
        vm.search("kotlin")
        assertThat(vm.state.value.loading).isFalse()
    }

    @Test
    fun synchronousWorkInsideRunTest() = runTest(dispatcher) {
        val actual = "hello".uppercase()
        assertEquals("HELLO", actual)
    }
}
