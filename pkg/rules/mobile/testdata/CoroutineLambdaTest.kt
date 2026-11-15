package demo

import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Test

class CoroutineLambdaTest {
    @Test
    fun localDispatcherUnderTrailingLambda() = runTest {
        val dispatcher = StandardTestDispatcher(testScheduler)
        val vm = SearchViewModel(FakeRepo(), dispatcher)
        vm.search("kotlin")
        assertEquals(State.Loaded, vm.state.value)
    }

    @Test
    fun blockBodyTrailingLambda() {
        runTest {
            val dispatcher = StandardTestDispatcher(testScheduler)
            val vm = SearchViewModel(FakeRepo(), dispatcher)
            vm.reload()
            assertEquals(State.Unloaded, vm.state.value)
        }
    }

    @Test
    fun emptyParenFormForParity() {
        runTest() {
            val dispatcher = StandardTestDispatcher(testScheduler)
            val vm = SearchViewModel(FakeRepo(), dispatcher)
            vm.refresh("parity", true)
            assertEquals(State.Loaded, vm.state.value)
        }
    }

    @Test
    fun advancedBeforeAssertion() = runTest {
        val dispatcher = StandardTestDispatcher(testScheduler)
        val vm = SearchViewModel(FakeRepo(), dispatcher)
        vm.search("kotlin")
        advanceUntilIdle()
        assertEquals(State.Loaded, vm.state.value)
    }

    @Test
    fun dispatcherNotInjectedIntoSubject() = runTest {
        val dispatcher = StandardTestDispatcher(testScheduler)
        val vm = SearchViewModel(FakeRepo())
        vm.search("kotlin")
        assertEquals(State.Loaded, vm.state.value)
    }

    @Test
    fun stateAssertedWithoutCallingSubject() = runTest {
        val dispatcher = StandardTestDispatcher(testScheduler)
        val vm = SearchViewModel(FakeRepo(), dispatcher)
        assertEquals(State.Loaded, vm.state.value)
    }

    @Test
    fun assertsAPlainValue() = runTest {
        val dispatcher = StandardTestDispatcher(testScheduler)
        val vm = SearchViewModel(FakeRepo(), dispatcher)
        vm.search("kotlin")
        assertEquals("kotlin", vm.lastQuery)
    }
}
