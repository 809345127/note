/**
 * ButtonManager - Similar to a Java class, manages button click handlers
 * @class
 */
class ButtonManager {
    constructor(outputElement) {
        // Instance property (like a Java field)
        this.name = "ButtonManager Instance";
        this.clickCount = 0;
        this.outputElement = outputElement;
    }

    /**
     * BROKEN: Regular function loses 'this' context when passed as callback
     * In Java terms: This is like passing a method reference that gets rebound
     */
    handleClickRegular(event) {
        this.clickCount++;

        // BUG: 'this' is NOT the ButtonManager instance!
        // When called via addEventListener, 'this' becomes the DOM button element
        const message = `❌ REGULAR FUNCTION:\n` +
            `   Expected 'this.name': "ButtonManager Instance"\n` +
            `   Actual 'this.name': ${this.name || 'undefined'}\n` +
            `   'this' is actually: ${this instanceof HTMLElement ? 'HTMLButtonElement' : this}\n` +
            `   Click count accessible? ${typeof this.clickCount === 'number' ? this.clickCount : 'NO'}\n`;

        console.error(message);
        this.logToPage(message);
    }

    /**
     * SAFE: Arrow function preserves 'this' from surrounding scope
     * In Java terms: This captures 'this' like a final variable in a closure
     */
    handleClickArrow = (event) => {
        this.clickCount++;

        // CORRECT: 'this' is the ButtonManager instance
        const message = `✅ ARROW FUNCTION:\n` +
            `   Expected 'this.name': "ButtonManager Instance"\n` +
            `   Actual 'this.name': ${this.name}\n` +
            `   'this' type: ${this.constructor.name}\n` +
            `   Click count: ${this.clickCount}\n`;

        console.log(message);
        this.logToPage(message);
    }

    /**
     * WORKS: Manual binding (pre-ES6 solution)
     * In Java terms: Explicitly binding 'this' like a method reference with a specific receiver
     */
    handleClickBound(event) {
        this.clickCount++;

        const message = `⚠️  MANUAL BIND:\n` +
            `   Expected 'this.name': "ButtonManager Instance"\n` +
            `   Actual 'this.name': ${this.name}\n` +
            `   'this' type: ${this.constructor.name}\n` +
            `   Click count: ${this.clickCount}\n` +
            `   (Works, but verbose - arrow functions are cleaner)\n`;

        console.log(message);
        this.logToPage(message);
    }

    /**
     * Utility method to log to both console and page
     */
    logToPage(message) {
        const timestamp = new Date().toLocaleTimeString();
        this.outputElement.textContent += `\n\n[${timestamp}] ${message}`;
    }
}

// =============================================================================
// APPLICATION INITIALIZATION (like a main() method in Java)
// =============================================================================

document.addEventListener('DOMContentLoaded', () => {
    const outputElement = document.getElementById('output');
    const buttonManager = new ButtonManager(outputElement);

    // Get button elements
    const btn1 = document.getElementById('btn1');
    const btn2 = document.getElementById('btn2');
    const btn3 = document.getElementById('btn3');

    // =========================================================================
    // ATTACHING EVENT LISTENERS - THE CRITICAL DIFFERENCE
    // =========================================================================

    // ❌ BROKEN: Regular function loses context
    // When clicked, 'this' inside handleClickRegular will be the button element,
    // NOT the buttonManager instance. This is a common bug!
    btn1.addEventListener('click', buttonManager.handleClickRegular);

    // ✅ SAFE: Arrow function preserves context
    // 'this' is lexically bound to buttonManager instance where it was defined
    btn2.addEventListener('click', buttonManager.handleClickArrow);

    // ⚠️  WORKS but VERBOSE: Manual binding (old-school solution)
    // Creates a new bound function. Inefficient if you need to remove listener later.
    btn3.addEventListener('click', buttonManager.handleClickBound.bind(buttonManager));

    // =============================================================================
    // SUMMARY FOR JAVA DEVELOPERS
    // =============================================================================
    console.log(`
================================================================================
JAVASCRIPT 'this' EXPLANATION FOR JAVA DEVELOPERS
================================================================================
    
In Java: 
    this.alwaysRefersToCurrentInstance();
    
In JavaScript:
    Regular functions: this.dependsOnHowYouCallIt()  // Dynamic binding
    Arrow functions:   this.capturedFromDefinition()  // Lexical binding (like Java)
    
ANALOGY:
    Regular function in JS is like a Java method that can be "re-targeted" 
    to a different instance at runtime.
    
    Arrow function in JS is like a Java lambda that closes over 'this' from 
    the enclosing scope and cannot be changed.
    
MODERN BEST PRACTICE:
    Use arrow functions for ALL callbacks, event handlers, and anonymous functions 
    in ES6+ codebases. They prevent 90% of 'this'-related bugs.
================================================================================
    `);
});