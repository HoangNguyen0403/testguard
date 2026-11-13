package demo

import com.google.common.truth.Truth.assertThat
import io.mockk.coEvery
import io.mockk.every
import io.mockk.mockk
import org.junit.Assert.assertEquals
import org.junit.Test

class EchoProvenancePositiveTest {
    @Test
    fun assertEqualsTokenFirst() {
        val repo = mockk<Repo>()
        every { repo.name() } returns "alice"
        val viewModel = ProfileViewModel(repo)
        assertEquals("alice", viewModel.name)
    }

    @Test
    fun assertEqualsSubjectFirst() {
        val repo = mockk<Repo>()
        every { repo.count() } returns 7
        val viewModel = ProfileViewModel(repo)
        assertEquals(viewModel.count, 7)
    }

    @Test
    fun kotestInfixEcho() {
        val repo = mockk<Repo>()
        coEvery { repo.fetchId() } returns 123
        val viewModel = ProfileViewModel(repo)
        viewModel.id shouldBe 123
    }

    @Test
    fun truthMatcherEcho() {
        val repo = mockk<Repo>()
        every { repo.status() } returns "ACTIVE"
        val viewModel = ProfileViewModel(repo)
        assertThat(viewModel.status).isEqualTo("ACTIVE")
    }

    @Test
    fun directStubReceiverEcho() {
        val repo = mockk<Repo>()
        every { repo.name() } returns "alice"
        assertEquals("alice", repo.name)
    }
}
