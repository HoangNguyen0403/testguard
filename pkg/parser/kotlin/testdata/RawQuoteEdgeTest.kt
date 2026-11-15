package probe

import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.Test

// Regression fixture for raw strings that end in a quote. Kotlin closes a raw
// string on the last three quotes of a maximal quote run, so surplus quotes
// belong to the content; the lexer must not treat them as a new string.
class RawQuoteEdgeTest {
    private val fourQuotes = """ends with quote""""
    private val windowsPath = """C:\Users\dev\"""
    private val regex = """\d+\s"(\w+)"""".toRegex()
    private val raw = """
        raw " quote, ""double"", $dollar ${listOf(")").size} and } brace \
    """.trimIndent()
    private val escaped = "say \"hi\" \\ \$notTemplate ${'$'} }"
    private val quote = '\''
    private val dq = '"'
    private val brace = '}'
    private val unicode = '\u007B'

    @Test
    fun fourQuoteLength() {
        assertEquals(16, fourQuotes.length)
    }

    @Test
    fun windowsPathLength() {
        assertEquals(13, windowsPath.length)
    }

    @Test
    fun regexWithQuotedFragment() {
        assertTrue(regex.containsMatchIn("12 \"ab\""))
    }

    @Test
    fun rawTemplateKeepsBraces() {
        assertTrue(raw.contains("}"))
    }

    @Test
    fun `name with ' apostrophe and ( paren`() {
        val s = "${if (dq == '"') "q" else "n"}${escaped.length}"
        assertEquals("q", s.take(1))
    }

    @Test
    fun whenExpressionOnChars() {
        val r = when (val c = brace) {
            '}' -> "close"
            '{' -> "open"
            else -> "other $c"
        }
        assertEquals("close", r)
    }
}
