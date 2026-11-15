package demo

import io.mockk.every
import io.mockk.mockk
import org.junit.Assert.assertEquals
import org.junit.Test

class MockEchoPositiveTest {
    @Test
    fun mockEchoDirect() {
        val repo = mockk<Repo>()
        every { repo.name() } returns "alice"
        val viewModel = ProfileViewModel(repo)
        assertEquals("alice", viewModel.name)
    }
}
