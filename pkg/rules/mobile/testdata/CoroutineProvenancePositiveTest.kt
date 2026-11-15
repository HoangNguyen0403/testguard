package demo

import com.google.common.truth.Truth.assertThat
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.runTest
import org.junit.Test

class CoroutineProvenancePositiveTest {
    private val scheduler = StandardTestDispatcher()

    @Test
    fun unnamedSubjectStateRead() = runTest(scheduler) {
        val vm = SearchViewModel(FakeRepo(), scheduler)
        vm.search("kotlin")
        assertThat(vm.state.value.results).isEmpty()
    }

    @Test
    fun namedArgumentInjection() = runTest(scheduler) {
        val vm = SearchViewModel(repo = FakeRepo(), dispatcher = scheduler)
        vm.search("kotlin")
        assertThat(vm.state.value.loading).isFalse()
    }
}
