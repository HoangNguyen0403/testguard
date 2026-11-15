package demo

import com.google.common.truth.Truth.assertThat
import io.kotest.matchers.shouldBe
import io.mockk.mockk
import io.mockk.verify
import org.junit.Ignore
import org.junit.Test
import kotlin.test.assertFailsWith

class KotlinAdapterPositiveTest {
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

    @Ignore("flaky")
    @Test
    fun ignored() {
        assertThat(Calc.add(1, 2)).isEqualTo(3)
    }
}
