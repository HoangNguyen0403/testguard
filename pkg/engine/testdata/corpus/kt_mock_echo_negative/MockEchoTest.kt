package demo

import io.mockk.every
import io.mockk.mockk
import org.junit.Assert.assertEquals
import org.junit.Test

class MockEchoNegativeTest {
    @Test
    fun mockEchoTransformed() {
        val repo = mockk<Repo>()
        every { repo.name() } returns "alice"
        val viewModel = ProfileViewModel(repo)
        assertEquals("ALICE", viewModel.name.uppercase())
    }
}
