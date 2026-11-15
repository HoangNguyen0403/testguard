package demo

import app.cash.turbine.test
import com.google.common.truth.Truth.assertThat
import io.mockk.every
import io.mockk.mockk
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.DisplayName
import org.junit.jupiter.api.Nested
import org.junit.jupiter.api.Test
import org.junit.jupiter.params.ParameterizedTest
import org.junit.jupiter.params.provider.ValueSource

class AdversarialTest {
    private val dispatcher = StandardTestDispatcher()
    private val repo = mockk<Repo>()

    // A string and a comment that look like tests must not become units.
    private val doc = "@Test fun fakeInString() { }"
    // @Test fun commentedOut() { }

    private fun buildUser(): User = User("alice")

    private fun verifyActive(user: User) {
        assertTrue(user.active)
    }

    // ---- correct tests: an assertion each, no rule finding ----

    @Test
    @DisplayName("adds two numbers")
    fun annotationBetweenTestAndFun() {
        assertEquals(3, Calc.add(1, 2))
    }

    @ParameterizedTest
    @ValueSource(ints = [1, 2, 3])
    fun parameterised(n: Int) {
        assertTrue(n > 0)
    }

    @Nested
    inner class WhenEmpty {
        @Test
        fun nestedScope() {
            assertThat(Cart().items).isEmpty()
        }
    }

    @Test
    fun advancesTheScheduler() = runTest(dispatcher) {
        val vm = SearchViewModel(repo, dispatcher)
        vm.search("kotlin")
        advanceUntilIdle()
        assertThat(vm.state.value.loading).isFalse()
    }

    @Test
    fun collectsWithTurbine() = runTest(dispatcher) {
        val vm = SearchViewModel(repo, dispatcher)
        vm.state.test {
            vm.search("kotlin")
            assertThat(awaitItem().loading).isTrue()
        }
    }

    @Test
    fun transformedStubIsNotAnEcho() {
        every { repo.name() } returns "alice"
        val viewModel = ProfileViewModel(repo)
        assertEquals("ALICE", viewModel.displayName)
    }

    @Test
    fun composeNodeAssertion() {
        composeRule.onNodeWithText("Hello").assertIsDisplayed()
    }

    @Test
    fun usesSameFileHelper() {
        verifyActive(buildUser())
    }

    // ---- hollow tests: each must be reported ----

    @Test
    fun onlyPrints() {
        println(Calc.add(1, 2))
    }

    @Test
    fun truthMockEcho() {
        val expectedName = "alice"
        every { repo.name() } returns expectedName
        val viewModel = ProfileViewModel(repo)
        assertThat(viewModel.name).isEqualTo(expectedName)
    }

    @Test
    fun staleStateNeverAdvanced() = runTest(dispatcher) {
        val vm = SearchViewModel(repo, dispatcher)
        vm.search("kotlin")
        assertThat(vm.state.value.results).isEmpty()
    }
}
