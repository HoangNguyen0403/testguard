package demo

import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.runTest
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Test

// Regression fixture for the Kotlin lexer: string templates with nested
// strings, char literals holding braces/quotes, nested block comments, and
// backtick identifiers. Each test must stay discovered with its class scope and
// keep its assertion, and the commented @Ignore must not hide the next test.
class LexicalEdgeTest {
    @Test
    fun charLiteralBrace() {
        val open = '{'
        assertEquals('{', open)
    }

    @Test
    fun charLiteralQuote() {
        val quote = '"'
        assertEquals('"', quote)
    }

    @Test
    fun templateWithNestedStringBrace() {
        val text = "${listOf("}").joinToString("{")} end"
        assertEquals("} end", text)
    }

    @Test
    fun afterTemplate() {
        assertEquals(3, add(1, 2))
    }

    /* @Test fun commented() { } /* nested */ @Ignore */
    // @Disabled
    @Test
    fun afterCommentedIgnore() {
        val user = current()
        check(user.name == "a") { "name mismatch" }
    }

    @Test
    fun `backtick name with { brace } and quote "`() {
        assertEquals(4, add(2, 2))
    }

    @Test
    fun afterBacktick() {
        assertEquals(5, add(2, 3))
    }
}
