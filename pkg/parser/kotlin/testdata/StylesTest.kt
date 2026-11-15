package demo

import com.google.common.truth.Truth.assertThat
import io.kotest.matchers.shouldBe
import io.mockk.coVerify
import io.mockk.every
import io.mockk.mockk
import io.mockk.verify
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.runTest
import org.junit.Ignore
import org.junit.Test
import kotlin.test.assertFailsWith
import kotlin.test.assertTrue

class StylesTest {
    private val dispatcher = StandardTestDispatcher()

    @Test
    fun truthAssertion() {
        assertThat(Calc.add(1, 2)).isEqualTo(3)
    }

    @Test
    fun kotestInfix() {
        Calc.add(1, 2) shouldBe 3
    }

    @Test
    fun mockkVerifyBlock() {
        val repo = mockk<Repo>(relaxed = true)
        Service(repo).save("a")
        verify { repo.save("a") }
    }

    @Test(expected = IllegalArgumentException::class)
    fun expectedException() {
        Calc.divide(1, 0)
    }

    @Test
    fun assertFails() {
        assertFailsWith<IllegalArgumentException> { Calc.divide(1, 0) }
    }

    @Test
    fun `backtick name with spaces`() {
        assertThat(Calc.add(2, 2)).isEqualTo(4)
    }

    @Test
    fun trulyEmpty() {
        Calc.add(1, 2)
    }

    @Test
    fun tautology() {
        assertTrue(true)
    }

    @Ignore("flaky")
    @Test
    fun ignored() {
        assertThat(Calc.add(1, 2)).isEqualTo(3)
    }

    @Test
    fun staleStateUnderStandardDispatcher() = runTest(dispatcher) {
        val vm = SearchViewModel(repo = mockk(relaxed = true), dispatcher = dispatcher)
        vm.search("kotlin")
        assertThat(vm.state.value.loading).isFalse()
    }

    @Test
    fun mockEcho() {
        val repo = mockk<Repo>()
        every { repo.name() } returns "alice"
        val viewModel = ProfileViewModel(repo)
        kotlin.test.assertEquals("alice", viewModel.name)
    }
}
