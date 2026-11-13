package demo

import io.mockk.every
import io.mockk.mockk
import org.junit.Assert.assertEquals
import org.junit.Test

class EchoProvenanceNegativeTest {
    @Test
    fun commentStubIsNotEvidence() {
        val repo = mockk<Repo>()
        // every { repo.count() } returns 3
        val actual = Calculator().add(1, 2)
        assertEquals(3, actual.value)
    }

    @Test
    fun quotedStubIsNotEvidence() {
        val repo = mockk<Repo>()
        val snippet = "every { repo.count() } returns 3"
        val actual = Calculator().add(1, 2)
        assertEquals(3, actual.value)
    }

    @Test
    fun unrelatedActualIsNotEvidence() {
        val repo = mockk<Repo>()
        every { repo.count() } returns 3
        val viewModel = ProfileViewModel(repo)
        val actual = Calculator().add(1, 2)
        assertEquals(3, actual.value)
    }

    @Test
    fun consumerWithoutMockInjectionIsNotEvidence() {
        val repo = mockk<Repo>()
        every { repo.count() } returns 3
        val viewModel = ProfileViewModel(OtherRepo())
        assertEquals(3, viewModel.count)
    }

    @Test
    fun matchingTokenWithoutSubjectIsNotEvidence() {
        val repo = mockk<Repo>()
        every { repo.count() } returns 3
        assertEquals(3, Calculator().add(1, 2))
    }

    @Test
    fun transformedChainIsNotEcho() {
        val repo = mockk<Repo>()
        every { repo.name() } returns "alice"
        val viewModel = ProfileViewModel(repo)
        assertEquals("ALICE", viewModel.name.uppercase())
    }

    @Test
    fun differentValueIsNotEcho() {
        val repo = mockk<Repo>()
        every { repo.name() } returns "alice"
        val viewModel = ProfileViewModel(repo)
        assertEquals("bob", viewModel.name)
    }

    @Test
    fun inequalityMatcherIsNotEcho() {
        val repo = mockk<Repo>()
        every { repo.name() } returns "alice"
        val viewModel = ProfileViewModel(repo)
        viewModel.name shouldNotBe "alice"
    }
}
